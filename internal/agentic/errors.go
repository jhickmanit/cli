// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package agentic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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
type ErrorNumber int

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

const (
	ErrorNumberUnknown     ErrorNumber = 1000
	ErrorNumberUsage       ErrorNumber = 1001
	ErrorNumberConfig      ErrorNumber = 1100
	ErrorNumberAuth        ErrorNumber = 1101
	ErrorNumberUnsupported ErrorNumber = 1200
	ErrorNumberPrompt      ErrorNumber = 1300
	ErrorNumberNotFound    ErrorNumber = 1404
	ErrorNumberConflict    ErrorNumber = 1409
	ErrorNumberRateLimited ErrorNumber = 1429
	ErrorNumberNetwork     ErrorNumber = 1500
	ErrorNumberInterrupted ErrorNumber = 1900
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

func RemoteError(message string, res *http.Response, err error) *CLIError {
	code, exitCode := remoteErrorCode(res)
	cliErr := NewError(code, exitCode, messageFromRemote(message, err), err)
	if res != nil {
		cliErr.RequestID = firstHeader(res.Header, "X-Request-Id", "X-Request-ID", "X-Ory-Request-Id", "X-Ory-Request-ID")
		cliErr.TraceID = firstHeader(res.Header, "Traceparent", "X-Trace-Id", "X-Trace-ID")
		cliErr.Details = map[string]any{"status_code": res.StatusCode}
	} else if err != nil {
		cliErr.Details = map[string]any{"cause": err.Error()}
	}
	return cliErr
}

func RemoteErrorWithBody(message string, res *http.Response, body []byte, err error) *CLIError {
	cliErr := RemoteError(message, res, err)
	if parsed := parseRemoteErrorBody(body); parsed != "" {
		cliErr.Message = parsed
	}
	return cliErr
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

func remoteErrorCode(res *http.Response) (ErrorCode, ExitCode) {
	if res == nil {
		return ErrorNetwork, ExitNetwork
	}
	switch res.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ErrorUsage, ExitUsage
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrorAuth, ExitAuth
	case http.StatusNotFound:
		return ErrorNotFound, ExitMissing
	case http.StatusConflict:
		return ErrorConflict, ExitConflict
	case http.StatusTooManyRequests:
		return ErrorRateLimited, ExitRateLimited
	}
	if res.StatusCode >= 500 {
		return ErrorNetwork, ExitNetwork
	}
	return ErrorNetwork, ExitNetwork
}

func messageFromRemote(message string, err error) string {
	if message != "" {
		if err != nil {
			return fmt.Sprintf("%s: %s", message, err)
		}
		return message
	}
	return messageFromError(err)
}

func parseRemoteErrorBody(body []byte) string {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return ""
	}

	var payload struct {
		Error struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	parts := make([]string, 0, 2)
	if payload.Error.Message != "" {
		parts = append(parts, payload.Error.Message)
	}
	if payload.Error.Reason != "" {
		parts = append(parts, payload.Error.Reason)
	}
	if payload.Message != "" {
		parts = append(parts, payload.Message)
	}
	if payload.Reason != "" {
		parts = append(parts, payload.Reason)
	}
	return strings.Join(parts, "\n")
}

func firstHeader(h http.Header, names ...string) string {
	for _, name := range names {
		if v := h.Get(name); v != "" {
			return v
		}
	}
	return ""
}

type ErrorEnvelope struct {
	SchemaVersion string      `json:"schema_version"`
	Code          ErrorCode   `json:"code"`
	CodeNumber    ErrorNumber `json:"code_number"`
	ExitCode      ExitCode    `json:"exit_code"`
	Message       string      `json:"message"`
	Prompt        string      `json:"prompt,omitempty"`
	RequestID     string      `json:"request_id,omitempty"`
	TraceID       string      `json:"trace_id,omitempty"`
	Details       any         `json:"details,omitempty"`
}

func WriteErrorJSON(w io.Writer, err error) error {
	cliErr := FromError(err)
	return json.NewEncoder(w).Encode(ErrorEnvelope{
		SchemaVersion: SchemaVersion,
		Code:          cliErr.Code,
		CodeNumber:    CodeNumber(cliErr.Code),
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
