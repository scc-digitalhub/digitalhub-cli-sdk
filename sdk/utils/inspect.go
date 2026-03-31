// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"

	"github.com/spf13/viper"
)

// GetConfigEntries returns key-value pairs for non-secret, persisted configuration fields.
func GetConfigEntries() map[string]string {
	return getEntries(false)
}

// GetCredentialEntries returns key-value pairs for secret, persisted credential fields.
func GetCredentialEntries() map[string]string {
	return getEntries(true)
}

func getEntries(secret bool) map[string]string {
	result := make(map[string]string)
	rt := reflect.TypeFor[Config]()

	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)

		if f.Tag.Get("persist") != "true" {
			continue
		}

		key := f.Tag.Get("vkey")
		if key == "" {
			continue
		}

		isSecret := f.Tag.Get("secret") == "true"
		if isSecret != secret {
			continue
		}

		val := viper.GetString(key)
		if val == "" {
			continue
		}

		result[key] = val
	}

	return result
}
