// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/services/crud"
)

func TestProjectsList(t *testing.T) {
	coreURL := os.Getenv("DHCORE_ENDPOINT")
	coreAPIVersion := os.Getenv("DHCORE_API_VERSION")
	coreToken := os.Getenv("DHCORE_ACCESS_TOKEN")

	if coreURL == "" || coreAPIVersion == "" || coreToken == "" {
		t.Skip("Missing env vars, skipping integration test.")
	}

	cfg := config.Config{
		Core: config.CoreConfig{
			BaseURL:     coreURL,
			APIVersion:  coreAPIVersion,
			AccessToken: coreToken,
		},
	}

	ctx := context.Background()

	svc, err := crud.NewCrudService(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to init sdk: %v", err)
	}

	elements, _, err := svc.ListAllPages(ctx, crud.ListRequest{
		ResourceRequest: crud.ResourceRequest{
			Project:  "gen-art2",
			Endpoint: "artifacts",
		},
		Params: map[string]string{},
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(elements) == 0 {
		t.Fatal("expected at least one project")
	}

	t.Logf("OK, found %d projects", len(elements))

	out, err := json.MarshalIndent(elements, "", "    ")
	if err != nil {
		t.Logf("Error serializing YAML: %v", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
