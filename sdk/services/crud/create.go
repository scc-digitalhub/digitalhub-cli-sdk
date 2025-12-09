// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

func (s *CrudService) Create(ctx context.Context, req CreateRequest) error {
	if req.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	if req.Endpoint != "projects" && req.Project == "" {
		return errors.New("project is mandatory for non-project resources")
	}

	var jsonMap map[string]any

	if req.FilePath != "" {
		// leggi YAML e converti in JSON -> map
		data, err := os.ReadFile(req.FilePath)
		if err != nil {
			return fmt.Errorf("failed to read YAML file: %w", err)
		}
		jsonBytes, err := yaml.YAMLToJSON(data)
		if err != nil {
			return fmt.Errorf("yaml to json failed: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
			return fmt.Errorf("failed to parse after JSON conversion: %w", err)
		}

		delete(jsonMap, "user")
		if req.Endpoint != "projects" {
			jsonMap["project"] = req.Project
		}
		if req.ResetID {
			delete(jsonMap, "id")
		}
	} else {
		// caso project senza file: usa solo name
		jsonMap = map[string]any{
			"name": req.Name,
		}
	}

	body, err := json.Marshal(jsonMap)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	url := s.http.BuildURL(req.Project, req.Endpoint, "", nil)
	_, _, err = s.http.Do(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	return nil
}
