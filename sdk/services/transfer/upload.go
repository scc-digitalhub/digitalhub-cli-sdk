// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Upload performs:
// - entity creation (if ID is empty) in the CREATED state with spec.path on S3
// - generate upload information to retrieve path
// - transition to UPLOADING
// - file/directory upload to s3://<bucket>/<project>/<resource>/<id>/...
// - transition to READY with files[] attached
func (s *TransferService) Upload(ctx context.Context, endpoint string, req UploadRequest) (*UploadResult, error) {
	if req.Input == "" {
		return nil, errors.New("missing required input file or directory")
	}
	if endpoint != "projects" && req.Project == "" {
		return nil, errors.New("project is mandatory for non-project resources")
	}

	// 1) Create the entity if ID is empty.
	entityID := req.ID
	if entityID == "" {
		entityIDPtr, err := s.createEntity(ctx, &req)
		if err != nil {
			return nil, fmt.Errorf("failed to create entity: %w", err)
		}
		entityID = *entityIDPtr
	}

	// 2) Retrieve the entity.
	getURL := s.http.BuildURL(req.Project, endpoint, entityID, nil)
	body, _, err := s.http.Do(ctx, "GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve entity info: %w", err)
	}
	var entity map[string]interface{}
	if err := json.Unmarshal(body, &entity); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 3) Check the status.
	status, ok := entity["status"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing or invalid status field")
	}
	state, _ := status["state"].(string)
	if state != "CREATED" {
		return nil, fmt.Errorf("entity is not in CREATED state, current state: %s", state)
	}

	// 4) Parse spec.path. If exists, it must use the s3 scheme.
	spec, ok := entity["spec"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing or invalid spec field")
	}
	pathStr, _ := spec["path"].(string)
	if pathStr != "" {
		parsedPath, err := ParsePath(pathStr)
		if err != nil {
			return nil, fmt.Errorf("invalid path in entity: %w", err)
		}
		if parsedPath.Scheme != "s3" {
			return nil, fmt.Errorf("only s3 scheme is supported for upload")
		}
	}

	// 5) Generate upload information to retrieve path.
	name, ok := entity["name"].(string)
	path, err := s.getUploadPath(ctx, endpoint, entityID, name, &req)
	if err != nil {
		return nil, fmt.Errorf("invalid path in upload info: %w", err)
	}
	parsedPath, err := ParsePath(*path)
	if err != nil {
		return nil, fmt.Errorf("invalid path in upload info: %w", err)
	}

	// 6) Helper: update status in Core (merge while preserving other fields).
	updateStatus := func(key string, updateData map[string]interface{}) error {
		existing, ok := entity[key].(map[string]interface{})
		if !ok {
			existing = map[string]interface{}{}
		}
		merged := MergeMaps(existing, updateData, MergeConfig{})
		entity[key] = merged

		payload, err := json.Marshal(entity)
		if err != nil {
			return fmt.Errorf("failed to marshal updated entity: %w", err)
		}
		putURL := s.http.BuildURL(req.Project, endpoint, entityID, nil)
		if _, _, err = s.http.Do(ctx, "PUT", putURL, payload); err != nil {
			return fmt.Errorf("failed to update entity status with data %v: %w", updateData, err)
		}
		return nil
	}

	// 7) Stato → UPLOADING
	if err := updateStatus("status", map[string]interface{}{"state": "UPLOADING"}); err != nil {
		return nil, err
	}

	// 8) Upload
	st, err := os.Stat(req.Input)
	if err != nil {
		_ = updateStatus("status", map[string]interface{}{"state": "ERROR"})
		return nil, fmt.Errorf("cannot access input: %w", err)
	}

	var files []map[string]interface{}
	ctxUp := ctx

	if st.IsDir() {
		_, files, err = UploadS3Dir(s.s3, ctxUp, parsedPath, req.Input, req.Verbose)
		if err != nil {
			_ = updateStatus("status", map[string]interface{}{"state": "ERROR"})
			return nil, fmt.Errorf("upload failed: %w", err)
		}
	} else {
		var targetKey string
		if strings.HasSuffix(parsedPath.Path, "/") {
			targetKey = filepath.ToSlash(filepath.Join(parsedPath.Path, st.Name()))
		} else {
			targetKey = parsedPath.Path
		}
		_, files, err = UploadS3File(s.s3, ctxUp, parsedPath.Host, targetKey, req.Input, req.Verbose)
		if err != nil {
			_ = updateStatus("status", map[string]interface{}{"state": "ERROR"})
			return nil, fmt.Errorf("upload failed: %w", err)
		}
	}

	// 9) State → READY + files
	if err := updateStatus("status", map[string]interface{}{
		"state": "READY",
		"files": files,
	}); err != nil {
		return &UploadResult{ArtifactID: entityID, Files: files}, fmt.Errorf("upload succeeded but failed to update status: %w", err)
	}

	return &UploadResult{ArtifactID: entityID, Files: files}, nil
}

func (s *TransferService) createEntity(ctx context.Context, req *UploadRequest) (*string, error) {

	// getRunKey func...retrieve the key from the run
	getRunKey := func() (string, error) {
		runID := viper.GetString(RunId)
		if runID == "" {
			return "", nil
		}

		url := s.http.BuildURL(req.Project, TranslateEndpoint("run"), runID, nil)
		bodyRun, _, err := s.http.Do(ctx, "GET", url, nil)
		if err != nil {
			return "", err
		}

		var run map[string]interface{}
		if err := json.Unmarshal(bodyRun, &run); err != nil {
			return "", err
		}

		if v, ok := run["key"].(string); ok && v != "" {
			return v, nil
		}
		return "", fmt.Errorf("run key not found in response")
	}

	// add a new relations in metadata
	addRelationship := func(entityMap map[string]interface{}, relType, dest string) {
		// Ensure metadata exists.
		meta, ok := entityMap["metadata"].(map[string]interface{})
		if !ok {
			meta = make(map[string]interface{})
			entityMap["metadata"] = meta
		}

		// Ensure relationships exists.
		rels, ok := meta["relationships"].([]map[string]interface{})
		if !ok {
			rels = []map[string]interface{}{}
		}

		// Add the new relationship.
		rels = append(rels, map[string]interface{}{
			"type": relType,
			"dest": dest,
		})

		meta["relationships"] = rels
	}

	runKey, err := getRunKey()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve run: %w", err)
	}
	if req.Name == "" {
		return nil, errors.New("name is required when creating a new entity")
	}
	bucket := req.Bucket
	if bucket == "" {
		bucket = "datalake" // Backward compatibility.
	}

	entityID := UUIDv4NoDash()

	path, err := s.getUploadPath(ctx, TranslateEndpoint(req.Resource), entityID, req.Name, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve upload path: %w", err)
	}

	entity := map[string]interface{}{
		"id":      entityID,
		"project": req.Project,
		"kind":    req.Resource,
		"name":    req.Name,
		"spec": map[string]interface{}{
			"path": *path,
		},
		"status": map[string]interface{}{
			"state": "CREATED",
		},
	}

	// Add lineage relationship
	if runKey != "" {
		addRelationship(entity, "produced_by", runKey)
	}

	payload, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity creation payload: %w", err)
	}

	endpoint := TranslateEndpoint(req.Resource)
	createURL := s.http.BuildURL(req.Project, endpoint, "", nil)

	if _, _, err = s.http.Do(ctx, "POST", createURL, payload); err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	return &entityID, nil
}

func (s *TransferService) getUploadPath(ctx context.Context, endpoint string, entityID string, entityName string, req *UploadRequest) (*string, error) {
	postURL := s.http.BuildURL(req.Project, endpoint, entityID, nil) + "/files/upload"
	params := url.Values{}
	if entityName != "" {
		params.Add("name", entityName)
	}
	if len(params) > 0 {
		postURL += "?" + params.Encode()
	}
	println("Upload URL:", postURL)
	body, _, err := s.http.Do(ctx, "POST", postURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve entity info: %w", err)
	}
	var uploadInfo map[string]interface{}
	if err := json.Unmarshal(body, &uploadInfo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	pathStr, ok := uploadInfo["path"].(string)
	if !ok {
		return nil, errors.New("missing or invalid path field")
	}
	return &pathStr, nil
}
