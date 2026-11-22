// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

// PrintMetrics mirrors MetricsService.PrintMetrics:
// - calls getContainerLog
// - retrieves status.metrics
// - prints "No metrics for this run." if there are no metrics
// - otherwise, pretty-prints the JSON.
func (s *RunService) PrintMetrics(ctx context.Context, req MetricsRequest) error {
	if req.Project == "" {
		return errors.New("project not specified")
	}
	if req.Resource == "" {
		return errors.New("endpoint not specified")
	}
	if req.ID == "" {
		return errors.New("resource id not specified")
	}

	containerLog, err := s.getContainerLog(ctx, req.RunResourceRequest, req.Container)
	if err != nil {
		return err
	}

	statusMap, ok := containerLog["status"].(map[string]interface{})
	if !ok {
		return errors.New("invalid log entry: missing status")
	}

	metricsVal := statusMap["metrics"]
	if metricsVal == nil {
		log.Println("No metrics for this run.")
		return nil
	}

	metricsSlice, ok := metricsVal.([]interface{})
	if !ok {
		return errors.New("invalid metrics format")
	}

	jsonData, err := json.Marshal(metricsSlice)
	if err != nil {
		return err
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, jsonData, "", "    "); err != nil {
		return err
	}
	fmt.Println(pretty.String())

	return nil
}

// getContainerLog mirrors the original logic:
// - GET /{endpoint}/{id}/logs
// - if the container is empty, derive the main container from spec.task
// - return the entry matching the selected container
func (s *RunService) getContainerLog(
	ctx context.Context,
	req RunResourceRequest,
	container string,
) (map[string]interface{}, error) {

	// 1) GET logs
	url := s.http.BuildURL(req.Project, req.Resource, req.ID, nil) + "/logs"
	body, status, err := s.http.Do(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("logs request failed (status %d): %w", status, err)
	}

	var logs []interface{}
	if err := json.Unmarshal(body, &logs); err != nil {
		return nil, fmt.Errorf("json parsing failed: %w", err)
	}

	// 2) Determine the container name (if not specified).
	containerName := container
	if containerName == "" {
		urlRes := s.http.BuildURL(req.Project, req.Resource, req.ID, nil)
		resBody, status, err := s.http.Do(ctx, "GET", urlRes, nil)
		if err != nil {
			return nil, fmt.Errorf("resource request failed (status %d): %w", status, err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(resBody, &m); err != nil {
			return nil, err
		}

		spec, ok := m["spec"].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid resource: missing spec")
		}
		task, ok := spec["task"].(string)
		if !ok {
			return nil, errors.New("invalid resource: missing task in spec")
		}

		idx := strings.Index(task, ":")
		if idx == -1 {
			return nil, errors.New("invalid task format in spec")
		}
		taskFormatted := strings.ReplaceAll(task[:idx], "+", "")

		containerName = fmt.Sprintf("c-%v-%v", taskFormatted, req.ID)
	}

	// 3) Find matching log entry
	for _, entry := range logs {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		statusVal, ok := entryMap["status"].(map[string]interface{})
		if !ok {
			continue
		}
		entryContainer, _ := statusVal["container"].(string)
		if entryContainer == containerName {
			return entryMap, nil
		}
	}

	return nil, fmt.Errorf("container not found")
}
