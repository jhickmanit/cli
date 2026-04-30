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
	require.Equal(t, "unable to list projects: connection refused", err.Message)
	require.Equal(t, map[string]any{"cause": "connection refused"}, err.Details)
}

func TestErrorRegistryIsUniqueAndComplete(t *testing.T) {
	seenCodes := map[ErrorCode]ErrorDefinition{}
	seenNumbers := map[ErrorNumber]ErrorCode{}
	for _, def := range ErrorRegistry {
		require.NotEmpty(t, def.Code)
		require.NotZero(t, def.CodeNumber)
		require.NotEmpty(t, def.Description)
		if previous, ok := seenCodes[def.Code]; ok {
			t.Fatalf("duplicate error code %q with exit codes %d and %d", def.Code, previous.ExitCode, def.ExitCode)
		}
		if previous, ok := seenNumbers[def.CodeNumber]; ok {
			t.Fatalf("duplicate error number %d for codes %q and %q", def.CodeNumber, previous, def.Code)
		}
		seenCodes[def.Code] = def
		seenNumbers[def.CodeNumber] = def.Code
	}

	expected := map[ErrorCode]ErrorDefinition{
		ErrorUnknown:     {Code: ErrorUnknown, CodeNumber: ErrorNumberUnknown, ExitCode: ExitUnknown},
		ErrorUsage:       {Code: ErrorUsage, CodeNumber: ErrorNumberUsage, ExitCode: ExitUsage},
		ErrorPrompt:      {Code: ErrorPrompt, CodeNumber: ErrorNumberPrompt, ExitCode: ExitUsage},
		ErrorConfig:      {Code: ErrorConfig, CodeNumber: ErrorNumberConfig, ExitCode: ExitConfig},
		ErrorAuth:        {Code: ErrorAuth, CodeNumber: ErrorNumberAuth, ExitCode: ExitAuth},
		ErrorNotFound:    {Code: ErrorNotFound, CodeNumber: ErrorNumberNotFound, ExitCode: ExitMissing},
		ErrorConflict:    {Code: ErrorConflict, CodeNumber: ErrorNumberConflict, ExitCode: ExitConflict},
		ErrorNetwork:     {Code: ErrorNetwork, CodeNumber: ErrorNumberNetwork, ExitCode: ExitNetwork},
		ErrorUnsupported: {Code: ErrorUnsupported, CodeNumber: ErrorNumberUnsupported, ExitCode: ExitUnsupported},
		ErrorRateLimited: {Code: ErrorRateLimited, CodeNumber: ErrorNumberRateLimited, ExitCode: ExitRateLimited},
		ErrorInterrupted: {Code: ErrorInterrupted, CodeNumber: ErrorNumberInterrupted, ExitCode: ExitInterrupted},
	}
	require.Equal(t, len(expected), len(seenCodes))
	for code, expectedDef := range expected {
		actual := seenCodes[code]
		require.Equal(t, expectedDef.Code, actual.Code)
		require.Equal(t, expectedDef.CodeNumber, actual.CodeNumber)
		require.Equal(t, expectedDef.ExitCode, actual.ExitCode)
		require.Equal(t, expectedDef.CodeNumber, CodeNumber(code))
	}
}
