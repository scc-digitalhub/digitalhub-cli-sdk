// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// taskToRunKind converts a task kind ("python+job", "python+job:task")
// to the corresponding run kind ("python+job:run").
func taskToRunKind(task string) string {
	task = strings.TrimSpace(task)
	if task == "" {
		return task
	}
	if i := strings.IndexByte(task, ':'); i >= 0 {
		// replace suffix with :run
		return task[:i] + ":run"
	}
	return task + ":run"
}

// Run creates a run while preserving the original logic.
func (s *RunService) Run(ctx context.Context, req RunRequest) error {
	if req.Project == "" {
		return errors.New("project not specified")
	}
	if req.TaskKind == "" {
		return errors.New("task kind not specified")
	}

	// IMPORTANT: preserve the original handling exactly.
	origTaskKind := req.TaskKind
	runKind := taskToRunKind(origTaskKind)

	// Resolve the function (returns its kind and key; the key is needed for spec).
	_, fnKey, err := s.resolveFunction(ctx, req.Project, req.FunctionID, req.FunctionName)
	if err != nil {
		return err
	}

	// Get or create the TASK using the ORIGINAL task kind (exact match).
	taskKey, err := s.getTaskKey(ctx, req.Project, fnKey, origTaskKind)
	if err != nil {
		taskKey, err = s.createTask(ctx, req.Project, fnKey, origTaskKind)
		if err != nil {
			return err
		}
	}

	// Build the spec: merge the input, then set the required fields.
	spec := map[string]interface{}{}
	for k, v := range req.InputSpec {
		spec[k] = v
	}
	spec["task"] = taskKey
	spec["function"] = fnKey
	spec["local_execution"] = false

	// RUN body: kind = runKind (e.g. "python+job:run").
	body := map[string]interface{}{
		"kind":    runKind,
		"project": req.Project,
		"spec":    spec,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	endpoint := req.ResolvedRunsEndpoint
	if endpoint == "" {
		endpoint = "runs"
	}
	url := s.http.BuildURL(req.Project, endpoint, "", nil)

	_, status, err := s.http.Do(ctx, "POST", url, data)
	if err != nil {
		return fmt.Errorf("run creation failed (status %d): %w", status, err)
	}
	return nil
}

func (s *RunService) resolveFunction(ctx context.Context, project, id, name string) (string, string, error) {
	var fn map[string]interface{}

	if id != "" {
		url := s.http.BuildURL(project, "functions", id, nil)
		b, status, err := s.http.Do(ctx, "GET", url, nil)
		if err != nil {
			return "", "", fmt.Errorf("get function by id failed (status %d): %w", status, err)
		}
		if err := json.Unmarshal(b, &fn); err != nil {
			return "", "", err
		}
	} else if name != "" {
		url := s.http.BuildURL(project, "functions", "", map[string]string{"name": name})
		b, status, err := s.http.Do(ctx, "GET", url, nil)
		if err != nil {
			return "", "", fmt.Errorf("get function by name failed (status %d): %w", status, err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return "", "", err
		}
		first, err := getFirstIfList(m)
		if err != nil {
			return "", "", err
		}
		fn = first
	} else {
		return "", "", errors.New("you must provide the name or ID of the function to run")
	}

	kind, ok1 := fn["kind"].(string)
	idVal, ok2 := fn["id"]
	nameVal, ok3 := fn["name"]
	if !ok1 || !ok2 || !ok3 {
		return "", "", errors.New("unable to obtain function key")
	}
	fnName := fmt.Sprint(nameVal)
	fnID := fmt.Sprint(idVal)
	return kind, fmt.Sprintf("%s://%s/%s:%s", kind, project, fnName, fnID), nil
}

func (s *RunService) getTaskKey(ctx context.Context, project, functionKey, taskKind string) (string, error) {
	params := map[string]string{"function": functionKey}
	url := s.http.BuildURL(project, "tasks", "", params)
	b, _, err := s.http.Do(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}

	if c, ok := m["content"].([]interface{}); ok {
		for _, it := range c {
			if tm, ok := it.(map[string]interface{}); ok {
				if k, ok := tm["kind"].(string); ok && k == taskKind {
					if idVal, ok := tm["id"]; ok {
						return fmt.Sprintf("%s://%s/%v", k, project, idVal), nil
					}
				}
			}
		}
	}
	return "", errors.New("unable to obtain task key")
}

func (s *RunService) createTask(ctx context.Context, project, functionKey, taskKind string) (string, error) {
	url := s.http.BuildURL(project, "tasks", "", nil)

	// EXACTLY as in the old code: use taskKind AS-IS.
	body := map[string]interface{}{
		"kind":    taskKind,
		"project": project,
		"spec": map[string]interface{}{
			"function": functionKey,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	b, status, err := s.http.Do(ctx, "POST", url, data)
	if err != nil {
		return "", fmt.Errorf("create task failed (status %d): %w", status, err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}
	first, err := getFirstIfList(m)
	if err != nil {
		return "", err
	}

	k, okk := first["kind"].(string)
	idVal, oki := first["id"]
	if !okk || !oki {
		return "", errors.New("unable to obtain task key")
	}
	return fmt.Sprintf("%s://%s/%v", k, project, idVal), nil
}

// Minimal copy of utils.GetFirstIfList to keep the SDK isolated.
func getFirstIfList(m map[string]interface{}) (map[string]interface{}, error) {
	if c, ok := m["content"].([]interface{}); ok && len(c) > 0 {
		if mm, ok := c[0].(map[string]interface{}); ok {
			return mm, nil
		}
		return nil, errors.New("invalid content element")
	}
	return m, nil
}
