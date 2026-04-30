// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

type ErrorDefinition struct {
	Code        ErrorCode `json:"code"`
	ExitCode    ExitCode  `json:"exit_code"`
	Description string    `json:"description"`
}

var ErrorRegistry = []ErrorDefinition{
	{
		Code:        ErrorUnknown,
		ExitCode:    ExitUnknown,
		Description: "Uncategorized failure.",
	},
	{
		Code:        ErrorUsage,
		ExitCode:    ExitUsage,
		Description: "Invalid flags, arguments, input shape, or validation.",
	},
	{
		Code:        ErrorPrompt,
		ExitCode:    ExitUsage,
		Description: "Command would need a prompt but interactivity is disabled.",
	},
	{
		Code:        ErrorConfig,
		ExitCode:    ExitConfig,
		Description: "Missing, unreadable, invalid, or incompatible CLI configuration.",
	},
	{
		Code:        ErrorAuth,
		ExitCode:    ExitAuth,
		Description: "Missing, expired, invalid, or unauthorized credentials.",
	},
	{
		Code:        ErrorNotFound,
		ExitCode:    ExitMissing,
		Description: "Requested resource does not exist.",
	},
	{
		Code:        ErrorConflict,
		ExitCode:    ExitConflict,
		Description: "Resource already exists or state conflict.",
	},
	{
		Code:        ErrorNetwork,
		ExitCode:    ExitNetwork,
		Description: "Transport failure, timeout, or remote service error.",
	},
	{
		Code:        ErrorUnsupported,
		ExitCode:    ExitUnsupported,
		Description: "Command is unsupported for selected target, profile, or capability.",
	},
	{
		Code:        ErrorRateLimited,
		ExitCode:    ExitRateLimited,
		Description: "Remote service throttled the request.",
	},
	{
		Code:        ErrorInterrupted,
		ExitCode:    ExitInterrupted,
		Description: "Command interrupted by signal or context cancellation.",
	},
}
