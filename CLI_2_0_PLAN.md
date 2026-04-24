# Ory CLI 2.0 Planning Proposal

This document captures the current 2.0 direction for the Ory CLI. It is intended to be a living design and execution plan that can be reviewed, edited, and picked up by another coding agent later.

## Status

Draft for review. Stage 0 validation is recorded in `CLI_2_0_STAGE_0_VALIDATION.md`.

No implementation work should be assumed complete unless a later section or linked issue/PR marks it complete.

## High-Level Goals

1. Make the CLI reliable for coding agents, CI systems, scripts, and other non-human callers.
2. Expand the CLI from an Ory Network-first tool into a unified Ory CLI that can operate against:
   - Ory Network
   - self-hosted Ory services
3. Wrap or expose the OSS CLI capabilities from Ory products instead of duplicating them ad hoc.
4. Keep the human CLI pleasant while making the machine contract strict and testable.
5. Preserve existing Ory Network workflows where practical, with compatibility aliases or migration guidance where behavior changes.

## Explicit Product Scope

The CLI should model two deployment targets:

- `network`: Ory Network projects and workspaces.
- `self-hosted`: user-managed Ory services.

OSS and Ory Enterprise License deployments should not be separate config models for the initial 2.0 work. They are both self-hosted from the CLI's perspective. OEL-only behavior should be represented later as feature/capability differences inside a self-hosted profile, not as a third target type.

## Current Repo Shape

The current root command is defined in `cmd/root.go` and registers mostly Ory Network commands from `cmd/cloudx`, plus a small Kratos JSONNet surface imported directly from `github.com/ory/kratos/cmd/jsonnet`.

The current Network command helper and config path live under `cmd/cloudx/client`. The current config file is `.ory-cloud.json`, and many command descriptions and flags say "Ory Network" directly. This makes Network the implicit default throughout command setup, target resolution, and help text.

Hydra and Kratos are already partially connected through Cobra context injection in `cmd/cloudx/client/client.go`. That file injects Network-backed Hydra and Kratos clients by resolving the selected Network project and using its project endpoint.

## Upstream OSS CLI Shape

The Ory product repositories already expose Go/Cobra command trees or command registration points:

- Hydra exposes root and recursive command registration. Its CLI includes create, get, list, update, delete, import, perform, introspect, revoke, migrate, serve, janitor, and version surfaces.
- Kratos exposes root and command registration for identities, jsonnet, hashers, migrate, serve, cleanup, courier, remote, and validation surfaces.
- Keto exposes command registration for relation tuples, namespace, migrate, server, check, expand, status, and version surfaces.
- Oathkeeper still needs a dedicated integration spike. The desired path is direct Go command integration if its packages expose a stable enough command API. If not, build an adapter around its library/client APIs instead of shelling out to an external binary.

This integration shape must be validated before implementation. The existence of Cobra constructors is not enough. Each product command surface needs to be checked for:

- direct `os.Exit` calls
- direct stdout/stderr writes
- package-level command variables or mutable global state
- assumptions about product-specific config files
- assumptions about process-global flags
- direct calls to deprecated fatal helpers
- whether endpoint/client injection can be done cleanly
- whether output and errors can be normalized without forking upstream code

Stage 0 validation found that the adapter-first fallback should be used for the primary 2.0 command surface. Hydra, Kratos, and Keto expose useful constructors and client/proto packages, but direct mounting inherits too many output, error, and process-control risks. Oathkeeper is not cleanly mountable and should remain outside the 2.0 release gate unless a small experimental adapter is explicitly chosen.

## Agentic CLI Requirements

The CLI should define and test a stable machine contract:

1. No command implementation should call `os.Exit`. Only `main` should convert an execution result to a process exit.
2. Commands should return typed errors with stable categories.
3. Stdout should contain only command result data.
4. Stderr should contain diagnostics, warnings, prompts, and progress.
5. `--output json` should produce valid JSON on success.
6. `--output json` should produce valid JSON on failure as well. JSON output mode implies JSON error formatting; avoid a public `--error-format` flag unless Stage 1 finds a strong need for it.
7. `--non-interactive` should disable prompts, browser opening, device flow waiting prompts, and other human-only behavior unless explicitly overridden.
8. `--agent` should be a convenience mode that implies:
   - `--output json`
   - `--non-interactive`
   - no color
   - no progress spinners
   - no automatic browser opening
9. Exit codes should be stable and documented.
10. Help and command metadata should be inspectable without scraping human-formatted text.
11. JSON success envelopes and error envelopes should include a `schema_version` field. Breaking output-shape changes require a new schema version, and the support window for old schema versions must be documented before 2.0 GA.
12. Error envelopes should preserve useful upstream diagnostics such as request IDs and trace IDs when available.

Suggested exit-code categories:

| Code | Meaning |
| ---: | --- |
| 0 | success |
| 1 | uncategorized error, reserved for genuinely unknown failures |
| 2 | usage or validation error |
| 3 | configuration error |
| 4 | authentication or authorization error |
| 5 | resource not found |
| 6 | conflict / already exists / state mismatch |
| 7 | network or remote service error, including timeouts |
| 8 | unsupported feature for selected target |
| 9 | rate-limited / throttled |
| 130 | interrupted |

Exact codes can change before implementation, but once shipped they should be treated as API.

Initial JSON error envelope shape:

```json
{
  "schema_version": "v1",
  "code": "resource_not_found",
  "exit_code": 5,
  "message": "resource was not found",
  "request_id": "...",
  "trace_id": "...",
  "details": {}
}
```

Initial non-interactive prompt error shape:

```json
{
  "schema_version": "v1",
  "code": "non_interactive_prompt_required",
  "exit_code": 2,
  "message": "this command requires interactive input",
  "prompt": "Enter a name for your project"
}
```

## Current Agentic Challenges

Known problems from the first repo pass:

- `cmd/root.go` maps all returned failures to `os.Exit(1)`, losing error category information.
- Upstream `cmdx` print helpers can call `os.Exit(1)` from format parsing and JSON pointer/path handling.
- Some code writes directly to stdout with `fmt.Print*` instead of using Cobra's configured output streams.
- Prompts are guarded command-by-command instead of through one shared interaction policy.
- `--quiet` is overloaded: it currently suppresses noise and changes output shape in ways that are not a full non-interactive machine contract.
- Config naming and help text are Network-specific.
- Some tests require Playwright browsers and fail before command behavior can be validated in a minimal environment.

## Command Model Direction

Product names do not need to be primary top-level commands for normal resource workflows.

The preferred user-facing model is verb/resource based:

```text
ory create project
ory list projects
ory get identity <id>
ory import identities ./identities.json
ory create oauth2-client
ory list oauth2-clients
ory create relation-tuple
ory check permission ...
ory validate identity ./identity.json
```

This matches much of the current CLI shape and is friendlier for agents because resources are the nouns agents need to manipulate.

Product names may still be useful as compatibility aliases, advanced escape hatches, or disambiguators:

```text
ory hydra ...
ory kratos ...
ory keto ...
ory oathkeeper ...
```

Those aliases should not be required for common workflows unless a command is inherently service-operational and ambiguous.

### Collision Analysis

The upstream and local CLIs share many generic verbs:

- `create`
- `get`
- `list`
- `update`
- `delete`
- `import`
- `perform`
- `introspect`
- `revoke`
- `validate`
- `serve`
- `migrate`
- `status`
- `version`

Generic verbs are not a problem if resource nouns are unique underneath them. For example, `create oauth2-client`, `create relation-tuple`, and `create project` can coexist.

The main collision area is operational commands where the target service matters:

- `serve`
- `migrate`
- `status`
- `cleanup`
- `courier`
- `janitor`

For these, avoid requiring product names where a domain noun is clearer:

```text
ory serve identity
ory serve oauth2
ory serve permissions
ory migrate identity sql ...
ory migrate oauth2 sql ...
ory migrate permissions ...
ory status identity
ory status oauth2
ory status permissions
```

Product aliases can still map directly to upstream command trees:

```text
ory kratos serve
ory hydra migrate sql
ory keto status
```

This gives humans a resource/domain-first CLI while preserving an escape hatch for users who already know the OSS product CLIs.

## Backward Compatibility Strategy

Backward compatibility is useful, but it is not an absolute requirement for 2.0.

The compatibility goal should be workflow compatibility, not exact command-path compatibility. If an old command's functionality exists through a clearer new command, it is acceptable to change the path as long as the migration is documented precisely.

Primary compatibility risks:

- GitHub Actions and CI scripts that invoke current commands.
- Internal automation that depends on current output formatting.
- Users who rely on `--quiet` behavior or first-column output.
- Users who parse human-formatted table output.

Recommended policy:

1. Preserve the underlying capability when possible.
2. Prefer the new command model when old paths conflict with the 2.0 design.
3. Provide an old-to-new command mapping table in release notes and docs.
4. Keep short-lived aliases only for high-traffic commands where the maintenance cost is low.
5. Do not preserve old behavior if it undermines the agentic output/error contract.
6. Make breaking changes explicit and easy to detect in CI.
7. Migrating from a v1 config creates a `default` profile pinned to `target: network` and sets it as active.
8. Commands should never infer target outside the resolver. The active profile, `--profile`, and explicit endpoint/profile environment variables are the only supported ways to change target.

Example migration table shape:

| 1.x command | 2.0 command | Notes |
| --- | --- | --- |
| `ory list projects` | `ory list projects` | likely unchanged for Network profiles |
| `ory create project` | `ory create project` | likely unchanged for Network profiles |
| `ory get identity <id>` | `ory get identity <id>` | target depends on active profile |
| `ory tunnel ...` | `ory tunnel ...` or `ory proxy tunnel ...` | needs validation |
| product-specific OSS command | resource/domain command | map during Stage 0 |

Common existing Network commands should continue to work after v1 migration because the migrated active profile is Network. New installs can choose either target during setup, but command behavior should still be resolver-driven rather than hardcoded to Network.

## Config Model

Introduce a config v2 profile system.

The config should support multiple named profiles and one active profile:

```json
{
  "version": "v2",
  "active_profile": "default",
  "profiles": {
    "default": {
      "target": "network",
      "network": {
        "selected_workspace": "...",
        "selected_project": "...",
        "console_url": "https://console.ory.sh",
        "api_url": "https://api.console.ory.sh"
      },
      "auth": {
        "type": "browser-session",
        "token_ref": "..."
      }
    },
    "local": {
      "target": "self-hosted",
      "self_hosted": {
        "endpoints": {
          "identity": {
            "public_url": "http://localhost:4433",
            "admin_url": "http://localhost:4434"
          },
          "oauth2": {
            "public_url": "http://localhost:4444",
            "admin_url": "http://localhost:4445"
          },
          "permissions": {
            "read_url": "http://localhost:4466",
            "write_url": "http://localhost:4467"
          },
          "oathkeeper": {
            "api_url": "http://localhost:4456"
          }
        }
      },
      "auth": {
        "type": "none"
      }
    }
  }
}
```

Use JSON as the on-disk format for 2.0. This aligns with the current `.ory-cloud.json` format and gives agents a stable, unambiguous parser target. Human-friendly workflows should be handled through commands such as `ory config edit`, `ory config set`, and `ory config view --output yaml`.

The migration should preserve the current `ORY_CONFIG_PATH` override and evaluate a neutral default path such as `$XDG_CONFIG_HOME/ory/config.json` on Linux and the equivalent platform-specific config locations on macOS and Windows.

### Profile Commands

Add or update profile/config commands:

```text
ory profile list
ory profile use <name>
ory profile create network <name>
ory profile create self-hosted <name>
ory profile get [name]
ory profile delete <name>
ory profile set-endpoint <profile> <service> <endpoint-name> <url>
ory config upgrade
```

Open question: whether these should be `profile` commands or live under `config profile`. Prefer `profile` if it remains compact and unambiguous.

## Target Resolution

Introduce an internal target resolver that every command uses.

Inputs:

- active profile
- `--profile`
- explicit endpoint flags
- Network project/workspace flags
- environment variables
- command requirements

Outputs:

- selected target type: `network` or `self-hosted`
- resolved service endpoint set
- auth strategy
- target applicability and required endpoint/auth checks
- machine-readable diagnostics if resolution fails

The resolver should be independent of Cobra where possible so it is easy to unit test.

## Feature Capabilities

Do not build a broad capability framework for initial 2.0 unless a concrete command requires it.

For initial 2.0, resolver checks should answer narrower questions:

- Does this command apply to the selected target type?
- Is the required endpoint configured?
- Is the required auth material available?
- Is a Network project/workspace required and selected?

Use exit code 8 for unsupported-feature responses driven by resolver checks. Reintroduce a fuller capability map when the first OEL-gated or feature-flagged command lands.

## Proposed Implementation Stages

Stage dependencies:

```text
Stage 0 ─┐
Stage 1 ─┼─> Stage 2 ─> Stage 3 ─> Stage 4 ─> Stage 6 ─> Stage 7
         │                         (Stage 5 optional / 2.1 candidate)
```

### Stage 0: Upstream Integration Validation

Goal: validate the core assumption that upstream OSS command trees can be mounted or adapted safely.

Work:

- Build a command-by-command integration matrix for Hydra, Kratos, Keto, and Oathkeeper.
- Classify each command as:
  - mount directly
  - mount with wrapper fixes
  - adapt through internal API/client calls
  - defer
  - exclude
- Identify commands that call `os.Exit`, write directly to stdout/stderr, or depend on process-global state.
- Identify which commands can accept injected clients/endpoints and which require config files, DSNs, or server runtime state.
- Identify commands whose output cannot be normalized without reimplementation.
- Decide whether product aliases should mount full upstream trees or only supported subtrees.
- Apply the fallback rule: if fewer than roughly 60% of prioritized upstream commands can be mounted directly or with trivial wrappers, pivot Stages 4 and 5 to an adapter-first strategy. In that strategy, the CLI calls upstream client/library APIs and reimplements thin local command surfaces. Product aliases mount only the subset that passes safety checks.

Acceptance criteria:

- A table exists with every relevant upstream command and its integration classification.
- The team knows which commands are safe to wrap before Stage 4 begins.
- The implementation path is chosen for Oathkeeper.
- Any required upstream patches or local adapter packages are identified.
- The 2.0 command map is updated with validated command paths.

### Stage 1: Agentic Execution Contract

Goal: establish the CLI behavior contract before adding more command surface.

Status: started on branch `cli-2-stage-1-agentic-contract`. The first slice moves process exit back to `main`, introduces typed CLI errors and JSON success/error envelopes, adds root `--output`, `--non-interactive`, `--agent`, and `--no-color` flags, and converts `ory version` into a representative in-process success/failure test target. Most existing command surfaces still use legacy `cmdx` output and error behavior until they are migrated command-by-command.

Work:

- Add an internal execution package that returns an execution result with an error category and exit code.
- Keep `os.Exit` in `main` only.
- Add structured error type(s), for example `CLIError`.
- Add root-level flags:
  - `--output`
  - `--non-interactive`
  - `--agent`
  - `--no-color`
- Separate quiet/noise behavior from machine output behavior.
- Add helper functions for writing success output and errors.
- Add lint/test coverage that catches direct `os.Exit` in command packages where feasible.

Acceptance criteria:

- New tests can execute commands in-process and observe stdout/stderr/errors without terminating the test process.
- Agent mode produces parseable JSON for representative success and failure commands.
- Prompts are rejected with a typed non-interactive error.

### Stage 2: Config v2 Profiles

Goal: make target selection explicit and extensible.

Work:

- Define config v2 structs and schema.
- Implement read/write support.
- Implement migration from `.ory-cloud.json` v1 to v2.
- Add `profile` commands.
- Preserve existing `ORY_CONFIG_PATH`, while considering a neutral path/name for future installs.
- Add environment variable overrides for selected profile and endpoints.

Acceptance criteria:

- Existing Network users can migrate without losing selected project/workspace/auth data.
- Self-hosted profiles can be created without Ory Network authentication.
- Commands can resolve a profile without reaching out to the network unless required.

### Stage 3: Target Resolver and Network Refactor

Goal: move current Network behavior onto the new target resolver without changing user-visible workflows unnecessarily.

Work:

- Replace direct `CommandHelper` config reads with resolver-driven state.
- Keep current Network resource commands working.
- Update help text from "Ory Network configuration file" to target-neutral language where appropriate.
- Ensure `--profile` and endpoint overrides are respected.
- Convert direct stdout/stderr writes to shared output helpers.

Acceptance criteria:

- Current Network commands still pass existing tests that do not require Playwright.
- Agent-mode tests cover representative Network commands using mocked clients.
- Old flags remain compatible or have explicit deprecation messages.

### Stage 4: OSS Command Integration for Hydra, Kratos, and Keto

Goal: expose OSS functionality through the unified command model.

Work:

- Mount or adapt Hydra commands into verb/resource paths.
- Mount or adapt Kratos commands into verb/resource paths.
- Mount or adapt Keto commands into verb/resource paths.
- Inject clients/endpoints using the target resolver.
- Prefer local adapter-first commands for the primary 2.0 surface, using upstream SDKs/protos/libraries rather than mounting full upstream Cobra trees.
- Add product aliases as optional compatibility/escape-hatch paths:
  - `ory hydra ...`
  - `ory kratos ...`
  - `ory keto ...`
- Normalize output and error handling for imported commands.

Acceptance criteria:

- Self-hosted profile can run representative identity, OAuth2, and permission commands.
- Network profile continues to route supported commands through Network project endpoints.
- Unsupported commands return a stable unsupported-feature error instead of failing mysteriously.

### Stage 5: Oathkeeper Integration Spike and Optional Implementation

Goal: determine the best Oathkeeper integration approach without making Oathkeeper a 2.0 release gate.

Work:

- Inspect Oathkeeper's Go packages and command layout.
- Decide whether direct Cobra mounting is viable.
- If viable, integrate like Hydra/Kratos/Keto.
- If not viable, build a focused adapter for the most important Oathkeeper CLI capabilities.
- Add Oathkeeper endpoint support to self-hosted profiles.
- Decide whether Oathkeeper ships behind an experimental flag in 2.0 or moves to 2.1.

Acceptance criteria:

- The chosen integration path is documented.
- If Oathkeeper ships in 2.0, at least one representative Oathkeeper command works against a configured self-hosted endpoint or local test server.
- Agent-mode output/error behavior matches the rest of the CLI.

### Stage 6: Command Metadata and Agent Discovery

Goal: make the CLI discoverable without scraping help text.

Work:

- Add a machine-readable command metadata command, for example:
  - `ory commands --output json`
  - `ory command get <path> --output json`
- Include command path, aliases, flags, required args, output schema hints, target requirements, and examples.
- Add generated docs for humans after the command model stabilizes.

Acceptance criteria:

- A coding agent can inspect available commands and determine required flags from JSON.
- Metadata output is stable enough to snapshot test.

### Stage 7: Compatibility, Docs, and Release Prep

Goal: finish the 2.0 transition cleanly.

Work:

- Document old-to-new command mappings.
- Add deprecation notices where command paths change.
- Update README and install docs.
- Add migration examples for Network and self-hosted users.
- Add release notes with agent-mode contract and exit-code table.
- Review shell completions and generated docs.

Acceptance criteria:

- Existing common Network workflows have either unchanged commands or documented aliases.
- Self-hosted workflows have first-class examples.
- Agentic behavior is documented and tested.

## Testing Strategy

Use layered tests:

1. Unit tests for config parsing, profile migration, target resolution, and error classification.
2. In-process Cobra execution tests with captured stdin/stdout/stderr.
3. Golden tests for JSON success/error output.
4. Mock API tests for Network and self-hosted service clients.
5. Optional browser/Playwright tests only for flows that truly need a browser.

Tests that require Playwright should be isolated behind a clear build tag or package boundary so basic CLI contract tests do not fail when browsers are not installed.

## Open Questions

1. What should the neutral config file name be long term: keep `.ory-cloud.json` for compatibility, introduce platform config paths such as `$XDG_CONFIG_HOME/ory/config.json`, or support both?
2. Should JSON schema version pinning use `--output json@v1`, `--schema-version v1`, `--output-version v1`, or another syntax?
3. Should product aliases be hidden from top-level help or visible as advanced commands?
4. How much of upstream command help text should be rewritten to fit the unified Ory CLI language?
5. Should local/self-hosted endpoint flags be global, profile-scoped, or both?
6. Should list commands support `--output ndjson` for streaming-safe agent consumption on large result sets?
7. Should structured diagnostics be a first-class flag such as `--diagnostics stderr`, or should existing verbose/debug semantics evolve to cover this?
8. What is the stability promise for command metadata output across minor versions?

## Recommended Decisions So Far

1. Use two target models only: `network` and `self-hosted`.
2. Treat OSS/OEL as self-hosted deployments; defer modeling their differences until a concrete command needs feature or license awareness.
3. Prefer verb/resource command paths over product-name namespaces for common workflows.
4. Keep product-name aliases for compatibility, advanced usage, and command integration fallback.
5. Build the agentic execution/output/error contract before expanding the command surface.
6. Migrated v1 configs create a `default` profile pinned to `network`; commands never infer target outside the resolver.
7. JSON success and error envelopes include `schema_version`; breaking shape changes require a new schema version.
8. `--output json` implies JSON-formatted errors; do not expose a public `--error-format` flag unless Stage 1 proves it is necessary.
9. On-disk config format is JSON in 2.0; human-friendly edits go through CLI commands such as `config edit`, `config set`, and formatted `config view`.
10. Oathkeeper is validated during Stage 0 but is not a 2.0 release gate unless the integration is low-risk.
11. Defer a broad capability map until a concrete OEL-gated or feature-flagged command requires it.
