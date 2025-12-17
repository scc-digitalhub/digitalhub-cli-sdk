// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/utils"
	"github.com/spf13/viper"
)

// Upload esegue:
// - creazione artefatto (se ID vuoto) in stato CREATED con spec.path su S3
// - transizione a UPLOADING
// - upload file/dir verso s3://<bucket>/<project>/<resource>/<id>/...
// - transizione a READY con files[] allegati
func (s *TransferService) Upload(ctx context.Context, endpoint string, req UploadRequest) (*UploadResult, error) {
	if req.Input == "" {
		return nil, errors.New("missing required input file or directory")
	}
	if endpoint != "projects" && req.Project == "" {
		return nil, errors.New("project is mandatory for non-project resources")
	}

	// getRunKey func...retrieve the key from the run
	getRunKey := func() (string, error) {
		runID := viper.GetString(utils.RunId)
		if runID == "" {
			return "", nil
		}

		// urlRun := utils.BuildCoreUrl(req.Project, utils.TranslateEndpoint("run"), runID, nil)
		// reqRun := utils.PrepareRequest("GET", urlRun, nil, viper.GetString(utils.DhCoreAccessToken))
		// bodyRun, err := utils.DoRequest(reqRun)
		url := s.http.BuildURL(req.Project, utils.TranslateEndpoint("run"), runID, nil)
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
	addRelationship := func(artifactMap map[string]interface{}, relType, dest string) {
		// assicurati che metadata esista
		meta, ok := artifactMap["metadata"].(map[string]interface{})
		if !ok {
			meta = make(map[string]interface{})
			artifactMap["metadata"] = meta
		}

		// assicurati che relationships esista
		rels, ok := meta["relationships"].([]map[string]interface{})
		if !ok {
			rels = []map[string]interface{}{}
		}

		// aggiungi nuova relazione
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

	// 1) Se ID vuoto: creare l'artefatto
	artifactID := req.ID
	if artifactID == "" {
		if req.Name == "" {
			return nil, errors.New("name is required when creating a new artifact")
		}
		bucket := req.Bucket
		if bucket == "" {
			bucket = "datalake" // retro-compat
		}

		st, err := os.Stat(req.Input)
		if err != nil {
			return nil, fmt.Errorf("cannot access input: %w", err)
		}

		artifactID = utils.UUIDv4NoDash()

		var path string
		if st.IsDir() {
			path = fmt.Sprintf("s3://%s/%s/%s/%s/", bucket, req.Project, req.Resource, artifactID)
		} else {
			path = fmt.Sprintf("s3://%s/%s/%s/%s/%s", bucket, req.Project, req.Resource, artifactID, st.Name())
		}

		entity := map[string]interface{}{
			"id":      artifactID,
			"project": req.Project,
			"kind":    req.Resource,
			"name":    req.Name,
			"spec": map[string]interface{}{
				"path": path,
			},
			"status": map[string]interface{}{
				"state": "CREATED",
			},
		}
		payload, err := json.Marshal(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal artifact creation payload: %w", err)
		}

		createURL := s.http.BuildURL(req.Project, endpoint, "", nil)

		if _, _, err = s.http.Do(ctx, "POST", createURL, payload); err != nil {
			return nil, fmt.Errorf("failed to create artifact: %w", err)
		}
	}

	// 2) Recupera l'artefatto
	getURL := s.http.BuildURL(req.Project, endpoint, artifactID, nil)
	body, _, err := s.http.Do(ctx, "GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve artifact info: %w", err)
	}
	var artifact map[string]interface{}
	if err := json.Unmarshal(body, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 3) Verifica stato
	status, ok := artifact["status"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing or invalid status field")
	}
	state, _ := status["state"].(string)
	if state != "CREATED" {
		return nil, fmt.Errorf("artifact is not in CREATED state, current state: %s", state)
	}

	// 4) Parse spec.path (deve essere s3)
	spec, ok := artifact["spec"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing or invalid spec field")
	}
	pathStr, _ := spec["path"].(string)
	parsedPath, err := utils.ParsePath(pathStr)
	if err != nil {
		return nil, fmt.Errorf("invalid path in artifact: %w", err)
	}
	if parsedPath.Scheme != "s3" {
		return nil, fmt.Errorf("only s3 scheme is supported for upload")
	}

	// Add lineage relationship
	if runKey != "" {
		addRelationship(artifact, "produced_by", runKey)
	}

	// 5) Helper: update status sul Core (merge preservando altri campi)
	updateStatus := func(key string, updateData map[string]interface{}) error {
		existing, ok := artifact[key].(map[string]interface{})
		if !ok {
			existing = map[string]interface{}{}
		}
		merged := utils.MergeMaps(existing, updateData, utils.MergeConfig{})
		artifact[key] = merged

		payload, err := json.Marshal(artifact)
		if err != nil {
			return fmt.Errorf("failed to marshal updated artifact: %w", err)
		}
		putURL := s.http.BuildURL(req.Project, endpoint, artifactID, nil)
		if _, _, err = s.http.Do(ctx, "PUT", putURL, payload); err != nil {
			return fmt.Errorf("failed to update artifact status with data %v: %w", updateData, err)
		}
		return nil
	}

	// 6) Stato → UPLOADING
	if err := updateStatus("status", map[string]interface{}{"state": "UPLOADING"}); err != nil {
		return nil, err
	}

	// 7) Upload
	st, err := os.Stat(req.Input)
	if err != nil {
		_ = updateStatus("status", map[string]interface{}{"state": "ERROR"})
		return nil, fmt.Errorf("cannot access input: %w", err)
	}

	var files []map[string]interface{}
	ctxUp := ctx

	if st.IsDir() {
		_, files, err = utils.UploadS3Dir(s.s3, ctxUp, parsedPath, req.Input, req.Verbose)
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
		_, files, err = utils.UploadS3File(s.s3, ctxUp, parsedPath.Host, targetKey, req.Input, req.Verbose)
		if err != nil {
			_ = updateStatus("status", map[string]interface{}{"state": "ERROR"})
			return nil, fmt.Errorf("upload failed: %w", err)
		}
	}

	// 8) Stato → READY + files
	if err := updateStatus("status", map[string]interface{}{
		"state": "READY",
		"files": files,
	}); err != nil {
		return &UploadResult{ArtifactID: artifactID, Files: files}, fmt.Errorf("upload succeeded but failed to update status: %w", err)
	}

	return &UploadResult{ArtifactID: artifactID, Files: files}, nil
}
