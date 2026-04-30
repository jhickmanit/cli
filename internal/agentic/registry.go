// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

type ErrorDefinition struct {
	Code        ErrorCode   `json:"code"`
	CodeNumber  ErrorNumber `json:"code_number"`
	ExitCode    ExitCode    `json:"exit_code"`
	Description string      `json:"description"`
}

var ErrorRegistry = []ErrorDefinition{
	{
		Code:        ErrorUnknown,
		CodeNumber:  ErrorNumberUnknown,
		ExitCode:    ExitUnknown,
		Description: "Uncategorized failure.",
	},
	{
		Code:        ErrorUsage,
		CodeNumber:  ErrorNumberUsage,
		ExitCode:    ExitUsage,
		Description: "Invalid flags, arguments, input shape, or validation.",
	},
	{
		Code:        ErrorPrompt,
		CodeNumber:  ErrorNumberPrompt,
		ExitCode:    ExitUsage,
		Description: "Command would need a prompt but interactivity is disabled.",
	},
	{
		Code:        ErrorConfig,
		CodeNumber:  ErrorNumberConfig,
		ExitCode:    ExitConfig,
		Description: "Missing, unreadable, invalid, or incompatible CLI configuration.",
	},
	{
		Code:        ErrorAuth,
		CodeNumber:  ErrorNumberAuth,
		ExitCode:    ExitAuth,
		Description: "Missing, expired, invalid, or unauthorized credentials.",
	},
	{
		Code:        ErrorNotFound,
		CodeNumber:  ErrorNumberNotFound,
		ExitCode:    ExitMissing,
		Description: "Requested resource does not exist.",
	},
	{
		Code:        ErrorConflict,
		CodeNumber:  ErrorNumberConflict,
		ExitCode:    ExitConflict,
		Description: "Resource already exists or state conflict.",
	},
	{
		Code:        ErrorNetwork,
		CodeNumber:  ErrorNumberNetwork,
		ExitCode:    ExitNetwork,
		Description: "Transport failure, timeout, or remote service error.",
	},
	{
		Code:        ErrorUnsupported,
		CodeNumber:  ErrorNumberUnsupported,
		ExitCode:    ExitUnsupported,
		Description: "Command is unsupported for selected target, profile, or capability.",
	},
	{
		Code:        ErrorRateLimited,
		CodeNumber:  ErrorNumberRateLimited,
		ExitCode:    ExitRateLimited,
		Description: "Remote service throttled the request.",
	},
	{
		Code:        ErrorInterrupted,
		CodeNumber:  ErrorNumberInterrupted,
		ExitCode:    ExitInterrupted,
		Description: "Command interrupted by signal or context cancellation.",
	},
}

var errorDefinitionsByCode = func() map[ErrorCode]ErrorDefinition {
	defs := make(map[ErrorCode]ErrorDefinition, len(ErrorRegistry))
	for _, def := range ErrorRegistry {
		defs[def.Code] = def
	}
	return defs
}()

func CodeNumber(code ErrorCode) ErrorNumber {
	if def, ok := errorDefinitionsByCode[code]; ok {
		return def.CodeNumber
	}
	return ErrorNumberUnknown
}
