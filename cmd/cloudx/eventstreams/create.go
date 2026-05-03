// Copyright © 2024 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package eventstreams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ory/cli/cmd/cloudx/client"
	"github.com/ory/cli/internal/agentic"
	cloud "github.com/ory/client-go"
	"github.com/ory/x/cmdx"
)

func NewCreateEventStreamCmd() *cobra.Command {
	c := streamConfig{}

	cmd := &cobra.Command{
		Use:   "event-stream [--project=PROJECT_ID] --type={sns,https} {--aws-iam-role-arn=arn:aws:iam::123456789012:role/MyRole --aws-sns-topic-arn=arn:aws:sns:us-east-1:123456789012:MyTopic, --https-endpoint=https://example.com/webhook}",
		Short: "Create a new event stream",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			h, err := client.NewCobraCommandHelper(cmd)
			if err != nil {
				return err
			}

			projectID, err := h.ProjectID()
			if err != nil {
				return err
			}

			if err := c.Validate(); err != nil {
				return err
			}
			stream, err := h.CreateEventStream(ctx, projectID, cloud.CreateEventStreamBody(c))
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(h.VerboseErrWriter, "Event stream created successfully!")
			return agentic.WriteRow(cmd, output(*stream))
		},
	}

	client.RegisterProjectFlag(cmd.Flags())
	client.RegisterWorkspaceFlag(cmd.Flags())
	cmdx.RegisterFormatFlags(cmd.Flags())

	registerStreamConfigFlags(cmd.Flags(), &c)

	return cmd
}
