// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

// Base shared by all operations on a "run-like" resource.
type RunResourceRequest struct {
	Project  string
	Resource string
	ID       string
}

// Request for logs and resource retrieval.
type LogRequest struct {
	RunResourceRequest
}

// Request for metrics (with an optional container).
type MetricsRequest struct {
	RunResourceRequest
	Container string
}

// Request for stopping a run.
type StopRequest struct {
	RunResourceRequest
}

// Request for resuming a run.
type ResumeRequest struct {
	RunResourceRequest
}

// Request for creating a run.
type RunRequest struct {
	Project      string
	TaskKind     string
	FunctionID   string
	FunctionName string
	InputSpec    map[string]interface{}

	// Endpoint for runs, already resolved by the adapter (e.g. "runs").
	ResolvedRunsEndpoint string
}
