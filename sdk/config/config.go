// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package config

import "net/http"

// Overall configuration passed to the SDK (no viper/INI here).
type Config struct {
	Core       CoreConfig
	S3         S3Config
	HTTPClient *http.Client // Optional: custom HTTP client for debugging/customization
}

type CoreConfig struct {
	BaseURL           string
	APIVersion        string
	AccessToken       string
	BasicAuthUsername string
	BasicAuthPassword string
}

type S3Config struct {
	AccessKey   string
	SecretKey   string
	AccessToken string
	Region      string
	EndpointURL string
}
