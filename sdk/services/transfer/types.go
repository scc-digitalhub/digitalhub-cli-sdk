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
	ID       string // opzionale; se vuoto -> crea nuovo artefatto
	Name     string // obbligatorio se ID vuoto (creazione)
	Input    string // file o directory locale (obbligatorio)
	Verbose  bool
	// Opzionale: override del bucket (default = "datalake" per compatibilità)
	Bucket string
}

type UploadResult struct {
	ArtifactID string
	Files      []map[string]interface{} // come in READY.status.files
}
