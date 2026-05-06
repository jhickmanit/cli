// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ory/cli/cmd/cloudx/client"
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
	require.Equal(t, internalagentic.ErrorNumberUsage, envelope.CodeNumber)
	require.Equal(t, internalagentic.ExitUsage, envelope.ExitCode)
	require.Contains(t, envelope.Message, "version accepts no arguments")
}

func TestAgenticPromptRequiredOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := execute(t.Context(), []string{"--agent", "create", "workspace"}, nil, &stdout, &stderr)

	require.Equal(t, int(internalagentic.ExitUsage), code)
	require.Empty(t, stdout.String())

	var envelope internalagentic.ErrorEnvelope
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
	require.Equal(t, internalagentic.SchemaVersion, envelope.SchemaVersion)
	require.Equal(t, internalagentic.ErrorPrompt, envelope.Code)
	require.Equal(t, internalagentic.ErrorNumberPrompt, envelope.CodeNumber)
	require.Equal(t, internalagentic.ExitUsage, envelope.ExitCode)
	require.Equal(t, "this command requires interactive input", envelope.Message)
	require.Equal(t, "Enter a name for your workspace", envelope.Prompt)
}

func TestClassifyCloudClientErrors(t *testing.T) {
	err := classifyExecutionError(NewRootCmd(), client.ErrNotAuthenticated)
	cliErr := internalagentic.FromError(err)

	require.Equal(t, internalagentic.ErrorAuth, cliErr.Code)
	require.Equal(t, internalagentic.ExitAuth, cliErr.ExitCode)

	err = classifyExecutionError(NewRootCmd(), client.ErrProjectNotSet)
	cliErr = internalagentic.FromError(err)

	require.Equal(t, internalagentic.ErrorUsage, cliErr.Code)
	require.Equal(t, internalagentic.ExitUsage, cliErr.ExitCode)
}

func TestAgenticAuthDoesNotStartInteractiveFlow(t *testing.T) {
	var stdout, stderr bytes.Buffer
	configPath := filepath.Join(t.TempDir(), "ory-cloud.json")

	code := execute(t.Context(), []string{"--agent", "auth", "--config", configPath}, nil, &stdout, &stderr)

	require.Equal(t, int(internalagentic.ExitAuth), code)
	require.Empty(t, stdout.String())
	require.NoFileExists(t, configPath)

	var envelope internalagentic.ErrorEnvelope
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
	require.Equal(t, internalagentic.SchemaVersion, envelope.SchemaVersion)
	require.Equal(t, internalagentic.ErrorAuth, envelope.Code)
	require.Equal(t, internalagentic.ErrorNumberAuth, envelope.CodeNumber)
	require.Equal(t, internalagentic.ExitAuth, envelope.ExitCode)
	require.Contains(t, envelope.Message, "interactive browser login")
}

func TestUserFacingCommandsDoNotExitProcess(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	disallowed := []string{
		"os.Exit(",
		"cmdx.Must(",
		"cmdx.Fatalf(",
		"cmdx.FailSilently(",
	}

	var violations []string
	err = filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, err error) error {
		require.NoError(t, err)
		if entry.IsDir() {
			if isProcessExitCheckIgnoredDir(repoRoot, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		require.NoError(t, err)
		if rel == "main.go" {
			return nil
		}

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, pattern := range disallowed {
			if strings.Contains(string(content), pattern) {
				violations = append(violations, rel+" contains "+pattern)
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, violations, "user-facing command code must return errors instead of terminating the process")
}

func isProcessExitCheckIgnoredDir(repoRoot, path string) bool {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return false
	}
	switch rel {
	case ".git",
		"cmd/clidoc",
		"cmd/dev",
		"cmd/pkg",
		"cmd/cloudx/e2e",
		"cmd/cloudx/testhelpers",
		"playwright-traces":
		return true
	default:
		return false
	}
}
