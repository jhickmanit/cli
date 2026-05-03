// Copyright © 2026 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
)

const (
	flagAgent  = "agent"
	flagOutput = "output"
)

func WantsJSONEnvelope(cmd *cobra.Command) bool {
	if flagBool(cmd, flagAgent) {
		return true
	}
	return flagString(cmd, flagOutput) == string(cmdx.FormatJSON)
}

func WriteTable(cmd *cobra.Command, out cmdx.Table) error {
	if WantsJSONEnvelope(cmd) {
		return WriteSuccessJSON(cmd.Root().OutOrStdout(), out.Interface())
	}

	cmdx.PrintTable(cmd, out)
	return nil
}

func flagBool(cmd *cobra.Command, name string) bool {
	f := cmd.Flag(name)
	if f == nil {
		return false
	}
	v, _ := strconv.ParseBool(f.Value.String())
	return v
}

func flagString(cmd *cobra.Command, name string) string {
	f := cmd.Flag(name)
	if f == nil {
		return ""
	}
	return f.Value.String()
}
