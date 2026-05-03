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

func TestWriteRowWrapsAgentOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("agent", true, "")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	require.NoError(t, WriteRow(cmd, testRow{value: "alpha"}))

	var envelope SuccessEnvelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, SchemaVersion, envelope.SchemaVersion)
	require.Equal(t, map[string]interface{}{"value": "alpha"}, envelope.Data)
}

func TestWriteJSONAbleWrapsAgentOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("agent", true, "")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	require.NoError(t, WriteJSONAble(cmd, testJSONAble{value: "alpha"}))

	var envelope SuccessEnvelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, SchemaVersion, envelope.SchemaVersion)
	require.Equal(t, map[string]interface{}{"value": "alpha"}, envelope.Data)
}

type testRow struct {
	value string
}

func (t testRow) Header() []string {
	return []string{"VALUE"}
}

func (t testRow) Columns() []string {
	return []string{t.value}
}

func (t testRow) Interface() interface{} {
	return map[string]string{"value": t.value}
}

type testJSONAble struct {
	value string
}

func (t testJSONAble) String() string {
	return t.value
}

func (t testJSONAble) Interface() interface{} {
	return map[string]string{"value": t.value}
}
