// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteErrorWithBodyMapsStatusAndMessage(t *testing.T) {
	res := &http.Response{
		StatusCode: http.StatusNotFound,
		Header: http.Header{
			"X-Request-Id": []string{"req-123"},
			"X-Trace-Id":   []string{"trace-123"},
		},
	}

	err := RemoteErrorWithBody("unable to get project", res, []byte(`{"error":{"message":"project not found","reason":"missing id"}}`), errors.New("remote failed"))

	require.Equal(t, ErrorNotFound, err.Code)
	require.Equal(t, ExitMissing, err.ExitCode)
	require.Equal(t, "project not found\nmissing id", err.Message)
	require.Equal(t, "req-123", err.RequestID)
	require.Equal(t, "trace-123", err.TraceID)
	require.Equal(t, map[string]any{"status_code": http.StatusNotFound}, err.Details)
}

func TestRemoteErrorWithoutResponseMapsToNetwork(t *testing.T) {
	err := RemoteError("unable to list projects", nil, errors.New("connection refused"))

	require.Equal(t, ErrorNetwork, err.Code)
	require.Equal(t, ExitNetwork, err.ExitCode)
	require.Equal(t, "unable to list projects", err.Message)
}

func TestErrorRegistryIsUniqueAndComplete(t *testing.T) {
	seen := map[ErrorCode]ExitCode{}
	for _, def := range ErrorRegistry {
		require.NotEmpty(t, def.Code)
		require.NotEmpty(t, def.Description)
		if previous, ok := seen[def.Code]; ok {
			t.Fatalf("duplicate error code %q with exit codes %d and %d", def.Code, previous, def.ExitCode)
		}
		seen[def.Code] = def.ExitCode
	}

	require.Equal(t, map[ErrorCode]ExitCode{
		ErrorUnknown:     ExitUnknown,
		ErrorUsage:       ExitUsage,
		ErrorPrompt:      ExitUsage,
		ErrorConfig:      ExitConfig,
		ErrorAuth:        ExitAuth,
		ErrorNotFound:    ExitMissing,
		ErrorConflict:    ExitConflict,
		ErrorNetwork:     ExitNetwork,
		ErrorUnsupported: ExitUnsupported,
		ErrorRateLimited: ExitRateLimited,
		ErrorInterrupted: ExitInterrupted,
	}, seen)
}
