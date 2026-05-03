// Copyright © 2026 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

type testTable struct {
	items []string
}

func (t testTable) Header() []string {
	return []string{"VALUE"}
}

func (t testTable) Table() [][]string {
	rows := make([][]string, len(t.items))
	for i, item := range t.items {
		rows[i] = []string{item}
	}
	return rows
}

func (t testTable) Interface() interface{} {
	return t.items
}

func (t testTable) Len() int {
	return len(t.items)
}

func TestWriteTableWrapsAgentOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("agent", true, "")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	require.NoError(t, WriteTable(cmd, testTable{items: []string{}}))

	var envelope SuccessEnvelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, SchemaVersion, envelope.SchemaVersion)
	require.Equal(t, []interface{}{}, envelope.Data)
}

func TestWriteTableKeepsDefaultTableOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("agent", false, "")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	require.NoError(t, WriteTable(cmd, testTable{items: []string{"alpha"}}))

	require.Contains(t, stdout.String(), "VALUE")
	require.Contains(t, stdout.String(), "alpha")
}
