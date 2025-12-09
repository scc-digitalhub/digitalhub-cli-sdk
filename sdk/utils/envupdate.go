// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// CheckUpdateEnvironment decides whether to refresh the environment:
// - missing/empty timestamp -> update
// - invalid timestamp       -> update
// - older than TTL          -> update
func CheckUpdateEnvironment() {
	const key = UpdatedEnvKey

	if viper.IsSet(IniSource) && viper.GetString(IniSource) == "env" {
		fmt.Printf("INI file has been created from enviromental variables...skip update\n")
		return
	}

	val := viper.GetString(key)
	isSet := viper.IsSet(key)
	fmt.Printf("Config freshness (%s): isSet=%v value=%q\n", key, isSet, val)

	if !isSet || val == "" {
		fmt.Println("Update: no timestamp.")
		updateEnvironment()
		return
	}

	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		fmt.Printf("Update: invalid timestamp (%v).\n", err)
		updateEnvironment()
		return
	}

	now := time.Now().UTC()
	age := now.Sub(t.UTC())
	ttl := time.Duration(outdatedAfterHours) * time.Hour

	if age >= ttl {
		fmt.Printf("Update: outdated (age %s ≥ TTL %s).\n", age, ttl)
		updateEnvironment()
		return
	}

	fmt.Printf("Fresh: age %s < TTL %s.\n", age, ttl)
}

// Fetch well-known, update Viper, bump timestamp, persist allowlisted keys.
func updateEnvironment() {
	fmt.Println("Updating environment…")
	baseEndpoint := viper.GetString(DhCoreEndpoint)
	if baseEndpoint == "" {
		fmt.Println("Skip: dhcore_endpoint is empty.")
		return
	}

	cfg, err := FetchConfig(baseEndpoint + "/.well-known/configuration")
	if err != nil {
		fmt.Printf("Config fetch failed: %v\n", err)
		return
	}
	for k, v := range cfg {
		viper.Set(k, ReflectValue(v))
	}

	oidc, err := FetchConfig(baseEndpoint + "/.well-known/openid-configuration")
	if err != nil {
		fmt.Printf("OpenID fetch failed: %v\n", err)
		return
	}
	for k, v := range oidc {
		viper.Set(k, ReflectValue(v))
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	viper.Set(UpdatedEnvKey, ts)
	fmt.Printf("Set %s=%s\n", UpdatedEnvKey, ts)

	env := viper.GetString(CurrentEnvironment)
	if env == "" {
		env = resolveEnvName()
	}
	if err := UpdateIniFromStruct(getIniPath(), env); err != nil {
		fmt.Printf("Persist failed: %v\n", err)
		return
	}
	fmt.Printf("Persisted to [%s].\n", env)
}

// Backward-compat wrapper.
func UpdateIniSectionFromViper(_ []string) error {
	env := viper.GetString(CurrentEnvironment)
	if env == "" {
		env = resolveEnvName()
	}
	if err := UpdateIniFromStruct(getIniPath(), env); err != nil {
		return fmt.Errorf("failed to save ini: %w", err)
	}
	fmt.Printf("Updated section [%s] in %s\n", env, getIniPath())
	return nil
}
