// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"github.com/spf13/cobra"

	"github.com/ory/cli/cmd/cloudx/client"
	"github.com/ory/cli/internal/agentic"
)

func NewListProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List your Ory Network projects.",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.NewCobraCommandHelper(cmd)
			if err != nil {
				return err
			}

			projects, err := h.ListProjects(cmd.Context(), h.WorkspaceID())
			if err != nil {
				return err
			}

			return agentic.WriteTable(cmd, &outputProjectCollection{projects})
		},
	}
	client.RegisterWorkspaceFlag(cmd.Flags())

	return cmd
}
