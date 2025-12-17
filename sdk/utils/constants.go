// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package utils

const (
	IniName                                 = ".dhcore.ini"
	IniSource                               = "ini_source"
	CurrentEnvironment                      = "current_environment"
	UpdatedEnvKey                           = "updated_environment"
	ApiLevelKey                             = "dhcore_api_level"
	DhCoreName                              = "dhcore_name"
	DhCoreIssuer                            = "dhcore_issuer"
	DhCoreClientId                          = "dhcore_client_id"
	DhCoreEndpoint                          = "dhcore_endpoint"
	DhCoreApiVersion                        = "dhcore_api_version"
	DhCoreAccessToken                       = "dhcore_access_token"
	DhCoreUser                              = "dhcore_user"
	DhCorePassword                          = "dhcore_password"
	DhCoreRefreshToken                      = "dhcore_refresh_token"
	Oauth2TokenEndpoint                     = "oauth2_token_endpoint"
	Oauth2UserinfoEndpoint                  = "oauth2_userinfo_endpoint"
	Oauth2AuthorizationEndpoint             = "oauth2_authorization_endpoint"
	Oauth2ScopesSupported                   = "oauth2_scopes_supported"
	Oauth2Issuer                            = "oauth2_issuer"
	Oauth2ResponseTypesSupported            = "oauth2_response_types_supported"
	Oauth2JwksUri                           = "oauth2_jwks_uri"
	Oauth2GrantTypesSupported               = "oauth2_grant_types_supported"
	Oauth2TokenEndpointAuthMethodsSupported = "oauth2_token_endpoint_auth_methods_supported"
	RunId                                   = "run_id"

	outdatedAfterHours = 1

	// API level the current version of the CLI was developed for
	MinApiLevel = 10

	// API level required for individual commands; 0 means no restriction
	LoginMin   = 10
	LoginMax   = 0
	CreateMin  = 10
	CreateMax  = 0
	ListMin    = 10
	ListMax    = 0
	GetMin     = 10
	GetMax     = 0
	UpdateMin  = 10
	UpdateMax  = 0
	DeleteMin  = 10
	DeleteMax  = 0
	StopMin    = 10
	StopMax    = 0
	ResumeMin  = 10
	ResumeMax  = 0
	LogMin     = 10
	LogMax     = 0
	MetricsMin = 10
	MetricsMax = 0
)

var DhCoreMap = map[string]string{
	"issuer":             DhCoreIssuer,
	"client_id":          DhCoreClientId,
	"dhcore_endpoint":    DhCoreEndpoint,
	"dhcore_api_version": DhCoreApiVersion,
	"access_token":       DhCoreAccessToken,
	"refresh_token":      DhCoreRefreshToken,
}

var Resources = map[string][]string{
	"artifacts": {"artifact"},
	"dataitems": {"dataitem"},
	"functions": {"function", "fn"},
	"models":    {"model"},
	"projects":  {"project"},
	"runs":      {"run"},
	"workflows": {"workflow"},
	"logs":      {"log"},
}
