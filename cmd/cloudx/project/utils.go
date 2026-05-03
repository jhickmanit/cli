// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/tidwall/sjson"

	"github.com/ory/cli/internal/agentic"
	cloud "github.com/ory/client-go"
)

func prefixConfig(prefix string, s []string) []string {
	for k := range s {
		s[k] = prefix + s[k]
	}
	return s
}

func prefixIdentityConfig(s []string) []string {
	return prefixConfig("/services/identity/config", s)
}

func prefixPermissionConfig(s []string) []string {
	return prefixConfig("/services/permission/config", s)
}

func prefixOAuth2Config(s []string) []string {
	return prefixConfig("/services/oauth2/config", s)
}

func prefixFileConfig(prefix string, configs []json.RawMessage) ([]json.RawMessage, error) {
	for k := range configs {
		raw, err := sjson.SetRawBytes(json.RawMessage("{}"), prefix, configs[k])
		if err != nil {
			return nil, err
		}
		configs[k] = raw
	}
	return configs, nil
}

func prefixFileIdentityConfig(configs []json.RawMessage) ([]json.RawMessage, error) {
	return prefixFileConfig("services.identity.config", configs)
}

func prefixFilePermissionConfig(configs []json.RawMessage) ([]json.RawMessage, error) {
	return prefixFileConfig("services.permission.config", configs)
}

func prefixFileOAuth2Config(configs []json.RawMessage) ([]json.RawMessage, error) {
	return prefixFileConfig("services.oauth2.config", configs)
}

func prefixFileNop(s []json.RawMessage) ([]json.RawMessage, error) {
	return s, nil
}

func outputFullProject(cmd *cobra.Command, p *cloud.SuccessfulProjectUpdate) error {
	return agentic.WriteRow(cmd, (*outputProject)(&p.Project))
}

func outputIdentityConfig(cmd *cobra.Command, p *cloud.SuccessfulProjectUpdate) error {
	return agentic.WriteJSONAble(cmd, outputConfig(p.Project.Services.Identity.Config))
}

func outputPermissionConfig(cmd *cobra.Command, p *cloud.SuccessfulProjectUpdate) error {
	return agentic.WriteJSONAble(cmd, outputConfig(p.Project.Services.Permission.Config))
}

func outputOAuth2Config(cmd *cobra.Command, p *cloud.SuccessfulProjectUpdate) error {
	return agentic.WriteJSONAble(cmd, outputConfig(p.Project.Services.Oauth2.Config))
}
