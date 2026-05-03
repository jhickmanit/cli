// Copyright © 2024 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package eventstreams

import (
	"github.com/spf13/cobra"

	"github.com/ory/cli/cmd/cloudx/client"
	"github.com/ory/cli/internal/agentic"
	"github.com/ory/x/cmdx"
)

func NewListEventStreamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event-streams",
		Args:  cobra.NoArgs,
		Short: "List your event streams",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := client.NewCobraCommandHelper(cmd)
			if err != nil {
				return err
			}

			projectID, err := h.ProjectID()
			if err != nil {
				return err
			}

			streams, err := h.ListEventStreams(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			return agentic.WriteTable(cmd, outputList(*streams))
		},
	}

	client.RegisterProjectFlag(cmd.Flags())
	client.RegisterWorkspaceFlag(cmd.Flags())
	cmdx.RegisterFormatFlags(cmd.Flags())
	return cmd
}
