// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CoreHTTP interface {
	BuildURL(project, resource, id string, params map[string]string) string
	Do(ctx context.Context, method, url string, data []byte) ([]byte, int, error)
}

type httpCore struct {
	httpClient *http.Client
	coreConfig CoreConfig
}

func NewHTTPCore(httpClient *http.Client, coreConfig CoreConfig) CoreHTTP {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &httpCore{httpClient: httpClient, coreConfig: coreConfig}
}

func (httpCore *httpCore) BuildURL(project, resource, id string, params map[string]string) string {
	base := fmt.Sprintf("%s/api/%s", httpCore.coreConfig.BaseURL, httpCore.coreConfig.APIVersion)
	if resource != "projects" && project != "" {
		base += "/-/" + project
	}
	base += "/" + resource
	if id != "" {
		base += "/" + id
	}
	first := true
	for k, v := range params {
		if v == "" {
			continue
		}
		if first {
			base += "?"
			first = false
		} else {
			base += "&"
		}
		base += fmt.Sprintf("%s=%s", k, v)
	}
	return base
}

func (httpCore *httpCore) Do(ctx context.Context, method, url string, data []byte) ([]byte, int, error) {
	var body io.Reader
	if data != nil {
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, err
	}
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// If access token is set, add Authorization header
	if tok := httpCore.coreConfig.AccessToken; tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	// If basic auth is set, add Basic Auth header
	if user := httpCore.coreConfig.BasicAuthUsername; user != "" {
		req.SetBasicAuth(user, httpCore.coreConfig.BasicAuthPassword)
	}

	resp, err := httpCore.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, rerr := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			if msg, ok := m["message"].(string); ok && msg != "" {
				return b, resp.StatusCode, fmt.Errorf("core responded with: %s - %s", resp.Status, msg)
			}
		}
		return b, resp.StatusCode, fmt.Errorf("core responded with: %s", resp.Status)
	}
	return b, resp.StatusCode, rerr
}
