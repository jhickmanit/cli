// Copyright © 2026 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"

	"github.com/ory/cli/internal/agentic"
)

func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	var cliErr *agentic.CLIError
	if errors.As(err, &cliErr) {
		return err
	}

	switch {
	case errors.Is(err, ErrNoConfig), errors.Is(err, ErrNoConfigQuiet), errors.Is(err, ErrNotAuthenticated), errors.Is(err, ErrReauthenticate):
		return agentic.NewError(agentic.ErrorAuth, agentic.ExitAuth, err.Error(), err)
	case errors.Is(err, ErrProjectNotSet):
		return agentic.UsageError(err)
	}

	return err
}
