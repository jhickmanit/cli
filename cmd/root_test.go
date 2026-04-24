// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	internalagentic "github.com/ory/cli/internal/agentic"
	"github.com/ory/x/cmdx"
)

func TestUsageTemplating(t *testing.T) {
	cmdx.AssertUsageTemplates(t, NewRootCmd())
}

func TestAgenticVersionOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := execute(t.Context(), []string{"--agent", "version"}, nil, &stdout, &stderr)

	require.Equal(t, int(internalagentic.ExitSuccess), code)
	require.Empty(t, stderr.String())

	var envelope struct {
		SchemaVersion string `json:"schema_version"`
		Data          struct {
			Version   string `json:"version"`
			GitHash   string `json:"git_hash"`
			BuildTime string `json:"build_time"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, internalagentic.SchemaVersion, envelope.SchemaVersion)
}

func TestAgenticErrorOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := execute(t.Context(), []string{"--output", "json", "version", "extra"}, nil, &stdout, &stderr)

	require.Equal(t, int(internalagentic.ExitUsage), code)
	require.Empty(t, stdout.String())

	var envelope internalagentic.ErrorEnvelope
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
	require.Equal(t, internalagentic.SchemaVersion, envelope.SchemaVersion)
	require.Equal(t, internalagentic.ErrorUsage, envelope.Code)
	require.Equal(t, internalagentic.ExitUsage, envelope.ExitCode)
	require.Contains(t, envelope.Message, "version accepts no arguments")
}
