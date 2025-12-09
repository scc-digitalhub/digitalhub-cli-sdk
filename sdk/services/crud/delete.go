// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

import (
	"context"
	"errors"
	"fmt"
)

func (s *CrudService) Delete(ctx context.Context, req DeleteRequest) error {
	if req.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	if req.Endpoint != "projects" && req.Project == "" {
		return errors.New("project is mandatory for non-project resources")
	}
	if req.ID == "" && req.Name == "" {
		return errors.New("you must specify id or name")
	}

	params := map[string]string{
		"cascade": "false",
	}
	if req.Cascade {
		params["cascade"] = "true"
	}

	id := req.ID
	if id == "" && req.Endpoint != "projects" {
		params["name"] = req.Name
		params["versions"] = "all"
	}

	url := s.http.BuildURL(req.Project, req.Endpoint, id, params)

	_, status, err := s.http.Do(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("delete failed (status %d): %w", status, err)
	}
	return nil
}
