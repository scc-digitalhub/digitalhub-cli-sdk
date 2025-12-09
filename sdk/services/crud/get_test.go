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

func TestGetByIDAndName(t *testing.T) {
	coreURL := os.Getenv("DHCORE_ENDPOINT")
	coreAPIVersion := os.Getenv("DHCORE_API_VERSION")
	coreToken := os.Getenv("DHCORE_ACCESS_TOKEN")

	if coreURL == "" || coreAPIVersion == "" || coreToken == "" {
		t.Skip("Missing env vars (DHCORE_ENDPOINT, DHCORE_API_VERSION, DHCORE_ACCESS_TOKEN), skipping integration test.")
	}

	endpoint := "artifacts"

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

	// Get first element to use its id/name for testing Get
	elements, _, err := svc.ListAllPages(ctx, crud.ListRequest{
		ResourceRequest: crud.ResourceRequest{
			Project:  "gen-art2",
			Endpoint: endpoint,
		},
		Params: map[string]string{
			"size": "1",
		},
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(elements) == 0 {
		t.Fatal("expected at least one resource to test Get")
	}

	first, ok := elements[0].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected element type: %#v", elements[0])
	}

	id := fmt.Sprint(first["id"])
	name := fmt.Sprint(first["name"])
	t.Logf("Using resource id=%s name=%s for Get test", id, name)

	// 2) GET by ID
	bodyByID, statusID, err := svc.Get(ctx, crud.GetRequest{
		ResourceRequest: crud.ResourceRequest{
			Project:  "gen-art2",
			Endpoint: endpoint,
		},
		ID:   id,
		Name: "",
	})
	if err != nil {
		t.Fatalf("Get by ID failed (status %d): %v", statusID, err)
	}

	// 3) GET by Name
	bodyByName, statusName, err := svc.Get(ctx, crud.GetRequest{
		ResourceRequest: crud.ResourceRequest{
			Project:  "gen-art2",
			Endpoint: endpoint,
		},
		ID:   "",
		Name: name,
	})
	if err != nil {
		t.Fatalf("Get by name failed (status %d): %v", statusName, err)
	}

	// 4) Confrontiamo i due risultati
	var mID, mName map[string]interface{}

	if err := json.Unmarshal(bodyByID, &mID); err != nil {
		t.Fatalf("json unmarshal (by ID) failed: %v", err)
	}
	if err := json.Unmarshal(bodyByName, &mName); err != nil {
		t.Fatalf("json unmarshal (by name) failed: %v", err)
	}

	// Se Get-by-name torna una "lista" con content, prendiamo il first (come fa l'adapter)
	if _, ok := mName["content"]; ok {
		var err error
		mName, err = getFirstFromList(mName)
		if err != nil {
			t.Fatalf("extract first element from name-response failed: %v", err)
		}
	}

	if fmt.Sprint(mID["id"]) != fmt.Sprint(mName["id"]) {
		t.Fatalf("id mismatch: byID=%v byName=%v", mID["id"], mName["id"])
	}
	if fmt.Sprint(mID["name"]) != fmt.Sprint(mName["name"]) {
		t.Fatalf("name mismatch: byID=%v byName=%v", mID["name"], mName["name"])
	}

	t.Logf("OK: Get by ID and Get by name returned the same resource")

	pretty, err := json.MarshalIndent(mID, "", "    ")
	if err == nil {
		fmt.Println("Resource from Get:")
		fmt.Println(string(pretty))
	}
}

// helper simile a utils.GetFirstIfList, ma locale al test per non importare utils
func getFirstFromList(m map[string]interface{}) (map[string]interface{}, error) {
	if c, ok := m["content"].([]interface{}); ok && len(c) > 0 {
		if mm, ok := c[0].(map[string]interface{}); ok {
			return mm, nil
		}
		return nil, fmt.Errorf("invalid content element")
	}
	return m, nil
}
