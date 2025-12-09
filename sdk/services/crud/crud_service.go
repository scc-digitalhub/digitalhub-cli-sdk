// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package crud

import (
	"context"
	"errors"

	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
)

type CrudService struct {
	http config.CoreHTTP
}

func NewCrudService(_ context.Context, conf config.Config) (*CrudService, error) {
	if conf.Core.BaseURL == "" || conf.Core.APIVersion == "" {
		return nil, errors.New("invalid core config")
	}
	return &CrudService{
		http: config.NewHTTPCore(nil, conf.Core),
	}, nil
}
