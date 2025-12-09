// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"context"

	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"

	"errors"
)

type RunService struct {
	http config.CoreHTTP
}

func NewRunService(ctx context.Context, conf config.Config) (*RunService, error) {
	if conf.Core.BaseURL == "" || conf.Core.APIVersion == "" {
		return nil, errors.New("invalid core config")
	}
	return &RunService{
		http: config.NewHTTPCore(nil, conf.Core),
	}, nil
}
