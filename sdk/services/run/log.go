// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"context"
	"errors"
	"fmt"
)

// GetLogs performs GET {base}/{project}/{endpoint}/{id}/logs
func (s *RunService) GetLogs(ctx context.Context, req LogRequest) ([]byte, int, error) {
	if req.Project == "" {
		return nil, 0, errors.New("project not specified")
	}
	if req.Resource == "" {
		return nil, 0, errors.New("endpoint not specified")
	}
	if req.ID == "" {
		return nil, 0, errors.New("id not specified")
	}

	url := s.http.BuildURL(req.Project, req.Resource, req.ID, nil) + "/logs"
	b, status, err := s.http.Do(ctx, "GET", url, nil)
	if err != nil {
		return nil, status, fmt.Errorf("get logs failed (status %d): %w", status, err)
	}
	return b, status, nil
}

// GetResource performs GET {base}/{project}/{endpoint}/{id}
// usato per leggere la risorsa run e derivare spec.task, ecc.
func (s *RunService) GetResource(ctx context.Context, req LogRequest) ([]byte, int, error) {
	if req.Project == "" {
		return nil, 0, errors.New("project not specified")
	}
	if req.Resource == "" {
		return nil, 0, errors.New("endpoint not specified")
	}
	if req.ID == "" {
		return nil, 0, errors.New("id not specified")
	}

	url := s.http.BuildURL(req.Project, req.Resource, req.ID, nil)
	b, status, err := s.http.Do(ctx, "GET", url, nil)
	if err != nil {
		return nil, status, fmt.Errorf("get resource failed (status %d): %w", status, err)
	}
	return b, status, nil
}
