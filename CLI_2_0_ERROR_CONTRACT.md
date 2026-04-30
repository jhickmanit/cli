# CLI 2.0 Error Contract

This document defines the planned CLI 2.0 error contract for agents, CI systems, and scripts.

The contract is intentionally separate from human-facing stderr text. Human text can improve over time. The fields below are the compatibility surface automation should rely on.

## Stability Rules

- `exit_code` values are stable after 2.0 GA.
- `code` values are stable after 2.0 GA.
- New `code` values may be added in minor releases.
- Removing a `code`, changing a `code` meaning, or changing a `code` to a different `exit_code` is breaking.
- JSON error fields may be added without a schema bump.
- Removing or renaming JSON error fields requires a new `schema_version`.
- CI workflows should branch on `code` first and `exit_code` second.
- CI workflows should not parse human stderr text.

## JSON Error Envelope

When JSON output is active through `--output json` or `--agent`, failures use this shape:

```json
{
  "schema_version": "v1",
  "code": "resource_not_found",
  "exit_code": 5,
  "message": "resource was not found",
  "prompt": "Enter a name for your project",
  "request_id": "req_...",
  "trace_id": "trace_...",
  "details": {}
}
```

Field meanings:

| Field | Required | Meaning |
| --- | --- | --- |
| `schema_version` | yes | Error envelope schema version. Initial value is `v1`. |
| `code` | yes | Stable machine-readable error code. |
| `exit_code` | yes | Process exit code returned by the CLI. |
| `message` | yes | Human-readable summary. Useful for logs, not stable enough for branching. |
| `prompt` | no | Prompt that would have been shown if interactivity were enabled. |
| `request_id` | no | Upstream request ID, when available. |
| `trace_id` | no | Upstream trace ID, when available. |
| `details` | no | Structured details. Shape depends on the error code and command. |

## Standard Error Codes

| Code | Exit code | Meaning |
| --- | ---: | --- |
| `unknown_error` | 1 | Uncategorized failure. |
| `usage_error` | 2 | Invalid flags, arguments, input shape, or validation. |
| `non_interactive_prompt_required` | 2 | Command would need a prompt but interactivity is disabled. |
| `configuration_error` | 3 | Missing, unreadable, invalid, or incompatible CLI configuration. |
| `authentication_error` | 4 | Missing, expired, invalid, or unauthorized credentials. |
| `resource_not_found` | 5 | Requested resource does not exist. |
| `conflict` | 6 | Resource already exists or state conflict. |
| `network_error` | 7 | Transport failure, timeout, or remote service error. |
| `unsupported_feature` | 8 | Command is unsupported for selected target, profile, or capability. |
| `rate_limited` | 9 | Remote service throttled the request. |
| `interrupted` | 130 | Command interrupted by signal or context cancellation. |

## CI Migration Notes

Current 1.x automation often treats every CLI failure as exit code `1` and may parse stderr text. In 2.0, automation should use JSON output and inspect stable fields.

Recommended pattern:

```sh
if ! out="$(ory --agent get project "$PROJECT_ID" 2>err.json)"; then
  code="$(jq -r '.code' err.json)"
  case "$code" in
    resource_not_found)
      echo "Project does not exist"
      ;;
    authentication_error)
      echo "Authentication failed"
      exit 4
      ;;
    rate_limited)
      echo "Retry later"
      exit 9
      ;;
    *)
      cat err.json >&2
      exit 1
      ;;
  esac
fi
```

Shell scripts that only need coarse handling can branch on `$?`, but `code` is preferred because multiple codes can share the same exit code. For example, `usage_error` and `non_interactive_prompt_required` both exit with `2`, but require different remediation.

## Breaking Change Notes

The following are expected 2.0 breaking changes for CI users:

- Remote API failures may no longer print exactly the same stderr text.
- JSON failure output is standardized when `--output json` or `--agent` is used.
- Remote errors may return specific exit codes instead of always exiting with `1`.
- Commands that would prompt return `non_interactive_prompt_required` under `--non-interactive` or `--agent`.

Release notes should include command-specific old-to-new mappings for high-traffic workflows as commands are migrated.
