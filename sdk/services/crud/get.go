// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

import (
	"context"
	"fmt"
)

func (s *CrudService) Get(ctx context.Context, req GetRequest) ([]byte, int, error) {
	params := map[string]string{}

	if req.ID == "" {
		if req.Name == "" {
			return nil, 0, fmt.Errorf("you must specify id or name")
		}
		params["name"] = req.Name
		params["versions"] = "latest"
	}

	url := s.http.BuildURL(req.Project, req.Endpoint, req.ID, params)
	return s.http.Do(ctx, "GET", url, nil)
}
