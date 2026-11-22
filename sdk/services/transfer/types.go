// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package transfer

type DownloadRequest struct {
	Project     string
	Resource    string
	ID          string
	Name        string
	Destination string
	Verbose     bool
}

type DownloadInfo struct {
	Filename string `json:"filename" yaml:"filename"`
	Size     int64  `json:"size"     yaml:"size"`
	Path     string `json:"path"     yaml:"path"`
}

// -------- Upload --------

type UploadRequest struct {
	Project  string
	Resource string
	ID       string // Optional; if empty, a new artifact is created.
	Name     string // Required when ID is empty (creation).
	Input    string // Local file or directory (required).
	Verbose  bool
	// Optional bucket override (default = "datalake" for compatibility).
	Bucket string
}

type UploadResult struct {
	ArtifactID string
	Files      []map[string]interface{} // Same as READY.status.files.
}
