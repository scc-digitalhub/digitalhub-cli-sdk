// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"context"
	"errors"
	"fmt"
)

// Resume performs POST {base}/{project}/{endpoint}/{id}/resume
// Ritorna body e status per far stampare lo stato all'adapter.
func (s *RunService) Resume(ctx context.Context, req ResumeRequest) ([]byte, int, error) {
	if req.Project == "" {
		return nil, 0, errors.New("project not specified")
	}
	if req.Resource == "" {
		return nil, 0, errors.New("endpoint not specified")
	}
	if req.ID == "" {
		return nil, 0, errors.New("id not specified")
	}

	url := s.http.BuildURL(req.Project, req.Resource, req.ID, nil) + "/resume"
	b, status, err := s.http.Do(ctx, "POST", url, nil)
	if err != nil {
		return nil, status, fmt.Errorf("resume request failed (status %d): %w", status, err)
	}
	return b, status, nil
}
