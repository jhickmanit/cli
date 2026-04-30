// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const SchemaVersion = "v1"

type ExitCode int

const (
	ExitSuccess     ExitCode = 0
	ExitUnknown     ExitCode = 1
	ExitUsage       ExitCode = 2
	ExitConfig      ExitCode = 3
	ExitAuth        ExitCode = 4
	ExitMissing     ExitCode = 5
	ExitConflict    ExitCode = 6
	ExitNetwork     ExitCode = 7
	ExitUnsupported ExitCode = 8
	ExitRateLimited ExitCode = 9
	ExitInterrupted ExitCode = 130
)

type ErrorCode string

const (
	ErrorUnknown     ErrorCode = "unknown_error"
	ErrorUsage       ErrorCode = "usage_error"
	ErrorPrompt      ErrorCode = "non_interactive_prompt_required"
	ErrorConfig      ErrorCode = "configuration_error"
	ErrorAuth        ErrorCode = "authentication_error"
	ErrorNotFound    ErrorCode = "resource_not_found"
	ErrorConflict    ErrorCode = "conflict"
	ErrorNetwork     ErrorCode = "network_error"
	ErrorUnsupported ErrorCode = "unsupported_feature"
	ErrorRateLimited ErrorCode = "rate_limited"
	ErrorInterrupted ErrorCode = "interrupted"
)

type CLIError struct {
	Code      ErrorCode `json:"code"`
	ExitCode  ExitCode  `json:"exit_code"`
	Message   string    `json:"message"`
	Prompt    string    `json:"prompt,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Details   any       `json:"details,omitempty"`
	Err       error     `json:"-"`
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *CLIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewError(code ErrorCode, exitCode ExitCode, message string, err error) *CLIError {
	return &CLIError{
		Code:     code,
		ExitCode: exitCode,
		Message:  message,
		Err:      err,
	}
}

func UsageError(err error) *CLIError {
	return NewError(ErrorUsage, ExitUsage, messageFromError(err), err)
}

func PromptRequired(prompt string) *CLIError {
	return &CLIError{
		Code:     ErrorPrompt,
		ExitCode: ExitUsage,
		Message:  "this command requires interactive input",
		Prompt:   prompt,
	}
}

func FromError(err error) *CLIError {
	if err == nil {
		return nil
	}

	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		if cliErr.Code == "" {
			cliErr.Code = ErrorUnknown
		}
		if cliErr.ExitCode == 0 {
			cliErr.ExitCode = ExitUnknown
		}
		if cliErr.Message == "" {
			cliErr.Message = messageFromError(cliErr.Err)
		}
		return cliErr
	}

	return NewError(ErrorUnknown, ExitUnknown, err.Error(), err)
}

type ErrorEnvelope struct {
	SchemaVersion string    `json:"schema_version"`
	Code          ErrorCode `json:"code"`
	ExitCode      ExitCode  `json:"exit_code"`
	Message       string    `json:"message"`
	Prompt        string    `json:"prompt,omitempty"`
	RequestID     string    `json:"request_id,omitempty"`
	TraceID       string    `json:"trace_id,omitempty"`
	Details       any       `json:"details,omitempty"`
}

func WriteErrorJSON(w io.Writer, err error) error {
	cliErr := FromError(err)
	return json.NewEncoder(w).Encode(ErrorEnvelope{
		SchemaVersion: SchemaVersion,
		Code:          cliErr.Code,
		ExitCode:      cliErr.ExitCode,
		Message:       cliErr.Message,
		Prompt:        cliErr.Prompt,
		RequestID:     cliErr.RequestID,
		TraceID:       cliErr.TraceID,
		Details:       cliErr.Details,
	})
}

type SuccessEnvelope struct {
	SchemaVersion string `json:"schema_version"`
	Data          any    `json:"data"`
}

func WriteSuccessJSON(w io.Writer, data any) error {
	return json.NewEncoder(w).Encode(SuccessEnvelope{
		SchemaVersion: SchemaVersion,
		Data:          data,
	})
}

func messageFromError(err error) string {
	if err == nil {
		return "command failed"
	}
	return fmt.Sprint(err)
}
