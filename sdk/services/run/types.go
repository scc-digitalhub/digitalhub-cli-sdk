// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

// Base comune per tutte le operazioni su una risorsa "run-like"
type RunResourceRequest struct {
	Project  string
	Endpoint string
	ID       string
}

// Request per logs e get resource
type LogRequest struct {
	RunResourceRequest
}

// Request per metrics (in più: container opzionale)
type MetricsRequest struct {
	RunResourceRequest
	Container string
}

// Request per stop
type StopRequest struct {
	RunResourceRequest
}

// Request per resume
type ResumeRequest struct {
	RunResourceRequest
}

// Request per creare un run
type RunRequest struct {
	Project      string
	TaskKind     string
	FunctionID   string
	FunctionName string
	InputSpec    map[string]interface{}

	// endpoint per i runs, già risolto dall'adapter (es. "runs")
	ResolvedRunsEndpoint string
}
