# CLI 2.0 Stage 0 Validation

This document records the Stage 0 validation pass for upstream Ory OSS CLI integration. It should be read with `CLI_2_0_PLAN.md`.

## Scope

Validated command integration assumptions for:

- Hydra, from module cache path `github.com/ory/hydra/v2@v2.3.1-0.20260324164235-735e0a30f7f2`
- Kratos, from module cache path `github.com/ory/kratos@v1.3.1-0.20260324164252-55fe178c4ea9`
- Keto, from module cache path `github.com/ory/keto@v0.14.1-0.20260324164236-ccb79cfc480c`
- Oathkeeper, cloned from `ory/oathkeeper` into `/tmp/ory-oathkeeper`

The validation looked for:

- command constructors and recursive registration APIs
- endpoint/client injection points
- direct `os.Exit`, `cmdx.Must`, `cmdx.Fatalf`, `Fatalf`, or `panic`
- direct stdout/stderr writes
- package-level command variables and `init` registration
- config and runtime assumptions that make embedding risky

## Summary

The adapter-first fallback should be considered active for 2.0.

Hydra, Kratos, and Keto expose useful command constructors or registration functions, but direct mounting is not enough to satisfy the agentic contract. Their command packages mix API calls with direct printing, fatal helpers, and format helpers that can terminate the process. Direct mounting is acceptable for selective product-alias escape hatches only after wrappers restrict or normalize risky behavior.

For primary 2.0 resource commands, use thin local adapters over upstream clients/libraries. This gives stable output envelopes, stable errors, and resolver-driven target selection without inheriting every upstream CLI behavior.

Oathkeeper should not be a 2.0 release gate. Its current command package is not cleanly mountable into this CLI.

## Integration Classifications

Classification meanings:

- `mount-with-wrapper`: usable through upstream Cobra command constructors or registration, but requires wrapper controls for output, errors, context, or command path.
- `adapter-first`: implement a local Ory CLI command surface that calls upstream clients/libraries directly.
- `defer`: do not include in first 2.0 implementation unless explicitly prioritized.
- `exclude`: not relevant to the unified CLI or unsafe to expose directly.

## Hydra

Hydra exposes `NewRootCmd(opts ...driver.OptionsModifier)` and `RegisterCommandRecursive(parent, opts...)` in `cmd/root.go`. It also exposes `cliclient.ClientContextKey` and `cliclient.OAuth2URLOverrideContextKey`, which this repo already uses in `cmd/cloudx/client/client.go`.

Direct mounting risks:

- `cmd/root.go` `Execute()` calls `os.Exit(1)`. Avoid `Execute`; only use constructors/registration.
- `cmd/version.go` writes with `fmt.Printf`.
- `cmd_perform_authorization_code.go` uses `cmdx.Must`, browser/listener flow messaging, and long-running local HTTP callbacks.
- `cmd_perform_client_credentials.go` writes usage to stdout with `fmt.Print`.
- `cmd_perform_device_flow.go` prints polling/user-code flow text.
- `cmd/server/*` and `cmd/serve_*` print banners and use fatal logger paths.
- `cmd/cli/error.go` can `panic`.
- `cmdx` printing supports formats that can call `os.Exit` on bad JSON path/pointer.

Hydra command matrix:

| Upstream command family | Relevant 2.0 resource/domain path | Classification | Notes |
| --- | --- | --- | --- |
| `create oauth2-client` | `ory create oauth2-client` | adapter-first | Client injection exists, but stable 2.0 output/error envelopes are easier with local adapter. |
| `get oauth2-client` | `ory get oauth2-client` | adapter-first | Existing command is close, but raw upstream output uses `cmdx` table/json behavior. |
| `list oauth2-clients` | `ory list oauth2-clients` | adapter-first | Needs stable pagination envelope and optional NDJSON later. |
| `update oauth2-client` | `ory update oauth2-client` | adapter-first | Local adapter should own validation and error mapping. |
| `delete oauth2-client` | `ory delete oauth2-client` | adapter-first | Local adapter should own success output. |
| `import oauth2-client` | `ory import oauth2-client` | adapter-first | File parsing and encryption key handling print direct errors upstream. |
| `create/get/delete/import jwk` | `ory create/get/delete/import jwk` | adapter-first | Upstream commands have direct stderr writes and input-format behavior to normalize. |
| `introspect token` | `ory introspect token` | adapter-first | API call is simple enough; local output envelope preferred. |
| `revoke token` | `ory revoke token` | adapter-first | Upstream prints explanatory errors directly. |
| `perform client-credentials` | `ory perform client-credentials` | adapter-first or defer | Non-interactive variant could be local; upstream usage and output are not agent-safe. |
| `perform authorization-code` | `ory perform authorization-code` | defer | Browser/listener flow conflicts with initial agentic contract. |
| `perform device-code` | `ory perform device-code` | defer | Interactive polling/user-code flow needs streaming/event design first. |
| `migrate sql/status` | `ory migrate oauth2 sql/status` | defer | Operational DSN/runtime command, not API-resource command. Needs separate non-interactive contract. |
| `serve admin/public/all` | `ory serve oauth2 ...` | defer | Long-running process with banners/fatal logger paths. |
| `janitor` | `ory janitor oauth2` or hidden product alias | defer | Operational command; direct usage output and DSN assumptions. |
| `version` | `ory version` | exclude local upstream mount | Use this CLI's version command, not product version printers. |
| `ory hydra ...` product alias | advanced compatibility path | mount-with-wrapper subset | Only mount safe subtrees after Stage 1 output/error controls exist. |

Hydra decision: prioritize adapter-first for resource commands. Product alias can mount a validated subset later, but should not be the primary implementation path.

## Kratos

Kratos exposes `NewRootCmd(driverOpts ...driver.RegistryOption)` in `cmd/root.go`. The command tree is modular: identities, jsonnet, hashers, migrate, serve, cleanup, courier, and remote. Kratos also exposes `cliclient.ClientContextKey`, which this repo already uses to inject Network-backed Kratos clients.

Direct mounting risks:

- `cmd/root.go` `Execute()` returns an int rather than exiting, which is better than Hydra/Keto.
- `cmd/clidoc/main.go` calls `os.Exit`, but clidoc is not part of the runtime CLI surface.
- `cmd/jsonnet/format.go` and `cmd/jsonnet/lint.go` use `cmdx.Must`; lint also calls `os.Exit(1)` for linter failures.
- `cmd/cliclient/migrate.go` writes usage with `fmt.Println`.
- hashers/load-test/calibrate commands emit human progress and timing text.
- identity commands are mostly close to mountable but still use `cmdx` output/error behavior.

Kratos command matrix:

| Upstream command family | Relevant 2.0 resource/domain path | Classification | Notes |
| --- | --- | --- | --- |
| `get identity` | `ory get identity` | adapter-first | Client injection exists; local adapter should own error envelope and multi-ID partial failure semantics. |
| `list identities` | `ory list identities` | adapter-first | Needs stable pagination envelope and resolver endpoints. |
| `delete identity` | `ory delete identity` | adapter-first | Local adapter should own success/no-content output. |
| `import identities` | `ory import identities` | adapter-first | File/stdin parsing and partial failures should be normalized. |
| `validate identity` | `ory validate identity` | mount-with-wrapper or adapter-first | Local validation may be better for machine output; upstream command prints human validation lines. |
| `jsonnet format/lint` | `ory format/lint jsonnet` | defer or adapter-first | Existing imported commands use `cmdx.Must` and `os.Exit`; current root already imports them but they are not agent-safe. |
| `hashers argon2 hash` | `ory hash password` or product alias | defer | Local utility command, not endpoint-backed; output/timing needs separate design. |
| `hashers argon2 calibrate/load-test` | product alias only | defer | Long-running/progress-heavy operational tooling. |
| `migrate sql` | `ory migrate identity sql` | defer | DSN/runtime operation; not a simple admin API command. |
| `serve` | `ory serve identity` | defer | Long-running server command. |
| `cleanup sql` | `ory cleanup identity sql` | defer | DSN/runtime operation. |
| `courier watch` | `ory courier watch` | defer | Runtime operation. |
| `remote status/version` | `ory status identity` | adapter-first or defer | Could be useful but should be resolver-backed and output-normalized. |
| `ory kratos ...` product alias | advanced compatibility path | mount-with-wrapper subset | Safer than Oathkeeper, but still not primary for agentic commands. |

Kratos decision: use adapter-first for identity API workflows. Keep direct mounting limited to product alias experiments or commands that are explicitly made agent-safe.

## Keto

Keto exposes `NewRootCmd(opts ...ketoctx.Option)` and package-level `RegisterCommandsRecursive` functions. It also has a useful context injection point in `cmd/client/grpc_client.go`: `ContextKeyDialFunc` can provide custom gRPC dialing and `ContextKeyTimeout` can provide timeouts.

Direct mounting risks:

- `cmd/root.go` `Execute()` calls `os.Exit(-1)` and prints with `fmt.Println`; avoid `Execute`.
- Relation tuple output conversion can `panic` through `MustNewProtoCollection`.
- Some commands print warnings/status text to stdout.
- `getRemote` emits fallback warnings to stderr when endpoint flags/env vars are absent.
- Migration/status commands are operational and print human progress.
- Current upstream commands use gRPC remotes, not HTTP URLs, so resolver must normalize endpoint forms carefully.

Keto command matrix:

| Upstream command family | Relevant 2.0 resource/domain path | Classification | Notes |
| --- | --- | --- | --- |
| `check` | `ory check permission` | adapter-first | Simple enough to implement locally over gRPC; upstream command is close but prints errors directly. |
| `expand` | `ory expand permission` | adapter-first | Needs stable tree output and empty-tree semantics. |
| `relation-tuple get/create/delete/delete-all` | `ory get/create/delete relation-tuple` | adapter-first | Context dial injection helps tests, but local output/error envelopes are still preferred. |
| `relation-tuple parse` | `ory parse relation-tuple` | mount-with-wrapper or adapter-first | Local parser may be small; no remote dependency. |
| `namespace validate` | `ory validate namespace` | adapter-first or defer | File/config validation output needs normalization. |
| `namespace opl-generate` | `ory generate opl` | defer | Utility command; lower priority for initial 2.0. |
| `migrate up/down/status` | `ory migrate permissions ...` | defer | Operational DB migration path, not API-resource command. |
| `serve` | `ory serve permissions` | defer | Long-running server command. |
| `status` | `ory status permissions` | adapter-first or defer | Could be implemented locally after endpoint model is settled. |
| `ory keto ...` product alias | advanced compatibility path | mount-with-wrapper subset | Safer than Oathkeeper; direct mount should be limited to validated subtrees. |

Keto decision: adapter-first for permissions resource commands, using upstream proto/API packages and gRPC dial behavior. Direct mounting is not needed for the primary 2.0 experience.

## Oathkeeper

Oathkeeper was cloned from `ory/oathkeeper` for this validation pass. Its command tree is materially less embeddable than Hydra/Kratos/Keto.

Direct mounting risks:

- `cmd/root.go` defines a package-level `RootCmd` global.
- Commands register through `init()` functions rather than exported constructors or recursive registration.
- `Execute()` calls `os.Exit(-1)`.
- Several commands call `cmdx.Must`, `cmdx.Fatalf`, or direct `os.Exit`.
- Rules and health commands print with `fmt.Println`.
- `cmd/helpers.go` uses `cmdx.Fatalf` if endpoint is missing.
- Rule API client lives under `internal/httpclient`, so this module cannot import it directly from `github.com/ory/cli`.
- Server command uses runtime/fatal logger paths and prints banners.

Oathkeeper command matrix:

| Upstream command family | Relevant 2.0 resource/domain path | Classification | Notes |
| --- | --- | --- | --- |
| `rules list/get` | `ory list/get access-rules` | adapter-first or defer | Cannot import internal generated client; local HTTP client or generated public client would be needed. |
| `health ready/alive` | `ory status oathkeeper` | adapter-first or defer | Upstream exits directly on unhealthy status. Local HTTP adapter is required for agent-safe output. |
| `credentials generate` | `ory generate credentials` | adapter-first or defer | Uses `cmdx.Must`; could be local utility later. |
| `serve` | `ory serve proxy` or product alias | defer | Long-running process with config/runtime assumptions. |
| `version` | `ory version` | exclude local upstream mount | Use unified CLI version. |
| `ory oathkeeper ...` product alias | experimental only | defer | Direct global `RootCmd` mounting is too risky for 2.0. |

Oathkeeper decision: do not make Oathkeeper a 2.0 release gate. If included at all in 2.0, ship a small experimental adapter for rules or health, not a mounted upstream command tree.

## Fallback Rule Result

The Stage 0 fallback rule from `CLI_2_0_PLAN.md` should trigger.

Fewer than 60% of prioritized upstream command families can be mounted directly while satisfying the agentic contract. Even the best candidates inherit `cmdx` format behavior and upstream output shapes. The safe implementation path is:

1. Build primary 2.0 commands as local adapter-first commands.
2. Reuse upstream clients, SDKs, proto packages, parsers, and small helper libraries where stable.
3. Avoid mounting full upstream root command trees in the main user path.
4. Add product aliases only for validated subtrees, and mark them advanced/compatibility-oriented.
5. Defer long-running operational commands until Stage 1 defines streaming/event output.

## Recommended Stage 4 Priority

Start with API-resource commands that are valuable for both Network and self-hosted targets:

1. Kratos identity commands:
   - `get identity`
   - `list identities`
   - `delete identity`
   - `import identities`
2. Hydra OAuth2 client commands:
   - `create oauth2-client`
   - `get oauth2-client`
   - `list oauth2-clients`
   - `update oauth2-client`
   - `delete oauth2-client`
3. Keto permission commands:
   - `check permission`
   - `create relation-tuple`
   - `get relation-tuple`
   - `delete relation-tuple`

Defer:

- `serve`
- database `migrate`
- browser or device `perform`
- Oathkeeper
- product full-root aliases

## Open Follow-Ups

1. Build a small prototype adapter command for one Hydra resource and one Kratos resource after Stage 1 output/error primitives exist.
2. Decide whether Stage 1 should replace or wrap `github.com/ory/x/cmdx` output helpers for local commands.
3. Decide whether product aliases should be hidden from generated help until their subtrees are validated.
4. Decide whether Oathkeeper should get a public generated client dependency or hand-written minimal HTTP calls if/when it enters scope.
