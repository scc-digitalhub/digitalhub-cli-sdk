// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

// usata embedded nelle altre request
type ResourceRequest struct {
	Project  string // obbligatorio per risorse != "projects"
	Resource string // "projects", "artifacts", ...
}

type CreateRequest struct {
	ResourceRequest

	Name     string
	FilePath string
	ResetID  bool
}

type DeleteRequest struct {
	ResourceRequest

	ID      string
	Name    string
	Cascade bool
}

type GetRequest struct {
	ResourceRequest

	ID   string
	Name string
}

type ListRequest struct {
	ResourceRequest

	Params map[string]string
}

type UpdateRequest struct {
	ResourceRequest

	ID   string
	Body []byte
}
