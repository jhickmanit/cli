// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ory/cli/cmd/cloudx"
	"github.com/ory/cli/cmd/cloudx/client"
	"github.com/ory/cli/cmd/cloudx/proxy"
	"github.com/ory/cli/internal/agentic"
	"github.com/ory/kratos/cmd/jsonnet"
	"github.com/ory/x/cmdx"
)

var commandTemplatingOnce sync.Once

const (
	flagOutput         = "output"
	flagNonInteractive = "non-interactive"
	flagAgent          = "agent"
	flagNoColor        = "no-color"
)

func NewRootCmd() *cobra.Command {
	c := &cobra.Command{
		Use:           "ory",
		Short:         "The Ory CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return applyAgenticFlags(cmd)
		},
	}
	c.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return agentic.UsageError(err)
	})

	registerAgenticFlags(c)

	c.AddCommand(devCommands...)
	c.AddCommand(
		cloudx.NewAuthCmd(),
		cloudx.NewCreateCmd(),
		jsonnet.NewFormatCmd(),
		jsonnet.NewLintCmd(),
		cloudx.NewDeleteCmd(),
		cloudx.NewGetCmd(),
		cloudx.NewUseCmd(),
		cloudx.NewListCmd(),
		cloudx.NewImportCmd(),
		cloudx.NewOpenCmd(),
		cloudx.NewPatchCmd(),
		cloudx.NewParseCmd(),
		cloudx.NewPerformCmd(),
		proxy.NewProxyCommand(),
		proxy.NewTunnelCommand(),
		cloudx.NewUpdateCmd(),
		cloudx.NewValidateCmd(),
		cloudx.NewRevokeCmd(),
		cloudx.NewIntrospectCmd(),
		cloudx.NewIsCmd(),
		NewVersionCmd(),
	)
	commandTemplatingOnce.Do(func() {
		cmdx.EnableUsageTemplating(c)
	})

	return c
}

func Execute() int {
	ctx := client.ContextWithClient(context.Background())
	return execute(ctx, nil, nil, nil, nil)
}

func execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	rootCmd := NewRootCmd()
	if args != nil {
		rootCmd.SetArgs(args)
	}
	if stdin != nil {
		rootCmd.SetIn(stdin)
	}
	if stdout != nil {
		rootCmd.SetOut(stdout)
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
	}
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		err = classifyExecutionError(rootCmd, err)
		if !errors.Is(err, cmdx.ErrNoPrintButFail) {
			if wantsJSONOutput(rootCmd) {
				_ = agentic.WriteErrorJSON(rootCmd.ErrOrStderr(), err)
			} else {
				_, _ = fmt.Fprintln(rootCmd.ErrOrStderr(), err)
			}
		}
		return int(agentic.FromError(err).ExitCode)
	}
	return int(agentic.ExitSuccess)
}

func registerAgenticFlags(c *cobra.Command) {
	c.PersistentFlags().String(flagOutput, "", "Set the machine output format. Currently supported: json.")
	c.PersistentFlags().Bool(flagNonInteractive, false, "Disable prompts and other interactive behavior.")
	c.PersistentFlags().Bool(flagAgent, false, "Enable agent-friendly defaults. Implies --output json, --non-interactive, and --no-color.")
	c.PersistentFlags().Bool(flagNoColor, false, "Disable colored output.")
}

func applyAgenticFlags(cmd *cobra.Command) error {
	if flagBool(cmd, flagAgent) {
		_ = cmd.Root().PersistentFlags().Set(flagOutput, string(cmdx.FormatJSON))
		_ = cmd.Root().PersistentFlags().Set(flagNonInteractive, "true")
		_ = cmd.Root().PersistentFlags().Set(flagNoColor, "true")
	}

	if flagBool(cmd, flagNoColor) {
		cmd.Root().Annotations = ensureAnnotations(cmd.Root().Annotations)
		cmd.Root().Annotations[flagNoColor] = "true"
	}

	if output := flagString(cmd, flagOutput); output != "" {
		if output != string(cmdx.FormatJSON) {
			return agentic.UsageError(fmt.Errorf("unsupported output format %q", output))
		}
		setFormatIfAvailable(cmd, output)
	}
	return nil
}

func classifyExecutionError(rootCmd *cobra.Command, err error) error {
	if errors.Is(err, cmdx.ErrNoPrintButFail) {
		return err
	}

	if rootCmd.SilenceUsage {
		return agentic.FromError(err)
	}

	return agentic.UsageError(err)
}

func wantsJSONOutput(cmd *cobra.Command) bool {
	if flagBool(cmd, flagAgent) {
		return true
	}
	return flagString(cmd, flagOutput) == string(cmdx.FormatJSON)
}

func setFormatIfAvailable(cmd *cobra.Command, output string) {
	if output != string(cmdx.FormatJSON) {
		return
	}
	if f := cmd.Flags().Lookup(cmdx.FlagFormat); f != nil && !f.Changed {
		_ = cmd.Flags().Set(cmdx.FlagFormat, output)
	}
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

func ensureAnnotations(in map[string]string) map[string]string {
	if in != nil {
		return in
	}
	return map[string]string{}
}
