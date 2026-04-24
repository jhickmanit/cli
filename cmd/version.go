// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/ory/cli/buildinfo"
	"github.com/ory/cli/internal/agentic"
	"github.com/ory/x/cmdx"

	"github.com/spf13/cobra"
)

type versionInfo struct {
	Version   string `json:"version"`
	GitHash   string `json:"git_hash"`
	BuildTime string `json:"build_time"`
}

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display this binary's version, build time, and git hash of this build",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			return agentic.UsageError(fmt.Errorf("version accepts no arguments"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			info := versionInfo{
				Version:   buildinfo.Version,
				GitHash:   buildinfo.GitHash,
				BuildTime: buildinfo.Time,
			}
			if flagString(cmd, flagOutput) == string(cmdx.FormatJSON) {
				return agentic.WriteSuccessJSON(cmd.Root().OutOrStdout(), info)
			}
			if flagBool(cmd, flagAgent) {
				return agentic.WriteSuccessJSON(cmd.Root().OutOrStdout(), info)
			}
			return printVersion(cmd, info)
		},
	}
}

func printVersion(cmd *cobra.Command, info versionInfo) error {
	_, err := fmt.Fprintf(cmd.Root().OutOrStdout(), "Version:    %s\nGit Hash:   %s\nBuild Time: %s\n", info.Version, info.GitHash, info.BuildTime)
	return err
}
