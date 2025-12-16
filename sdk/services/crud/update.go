// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

import (
	"context"
	"errors"
	"fmt"
)

func (s *CrudService) Update(ctx context.Context, req UpdateRequest) error {
	if req.Resource == "" {
		return errors.New("endpoint is required")
	}
	if req.ID == "" {
		return errors.New("id is required")
	}
	if req.Resource != "projects" && req.Project == "" {
		return errors.New("project is mandatory for non-project resources")
	}
	if len(req.Body) == 0 {
		return errors.New("empty body")
	}

	url := s.http.BuildURL(req.Project, req.Resource, req.ID, nil)
	_, status, err := s.http.Do(ctx, "PUT", url, req.Body)
	if err != nil {
		return fmt.Errorf("update failed (status %d): %w", status, err)
	}
	return nil
}
