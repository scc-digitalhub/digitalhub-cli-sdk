// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strconv"
)

func (s *CrudService) ListAllPages(ctx context.Context, req ListRequest) ([]interface{}, int, error) {
	var (
		elements   []interface{}
		currentPg  int
		totalPages int
	)

	pageParams := map[string]string{}
	if req.Params != nil {
		maps.Copy(pageParams, req.Params)
	}

	for {
		url := s.http.BuildURL(req.Project, req.Endpoint, "", pageParams)
		body, status, err := s.http.Do(ctx, "GET", url, nil)
		if err != nil {
			return nil, 0, err
		}
		if status != 200 {
			return nil, 0, fmt.Errorf("core responded with status %d", status)
		}

		m := map[string]interface{}{}
		if err := json.Unmarshal(body, &m); err != nil {
			return nil, 0, fmt.Errorf("json parsing failed: %w", err)
		}

		pageList, _ := m["content"].([]interface{})
		elements = append(elements, pageList...)

		if pg, ok := m["pageable"].(map[string]interface{}); ok {
			if v := reflect.ValueOf(pg["pageNumber"]); v.IsValid() {
				switch v.Kind() {
				case reflect.Float64:
					currentPg = int(v.Float())
				default:
					currentPg = 0
				}
			}
		}
		if tp, ok := m["totalPages"]; ok {
			switch v := tp.(type) {
			case float64:
				totalPages = int(v)
			default:
				totalPages = 1
			}
		} else {
			totalPages = 1
		}

		if currentPg >= totalPages-1 {
			break
		}
		pageParams["page"] = strconv.Itoa(currentPg + 1)
	}

	return elements, totalPages, nil
}
