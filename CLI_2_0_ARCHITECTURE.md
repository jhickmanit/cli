# CLI 2.0 Architecture Proposal

This document proposes a clean architecture for a CLI 2.0 implementation. It assumes the current Stage 1 branch is useful as a contract prototype, but not as the long-term structure for the 2.0 codebase.

The goal is to make the next implementation easy for a human or coding agent to extend without reintroducing hidden process exits, mixed output formats, implicit browser flows, or Network-only assumptions.

## Recommendation

Build the 2.0 CLI as a clean command layer and selectively bring over proven code from the current repository.

The current branch should be treated as a discovery branch. It validated the agentic execution contract and exposed the places where the existing code is tightly coupled to human-first behavior:

- command handlers print directly
- helpers combine config loading, target selection, auth, prompts, API clients, output, and side effects
- `cmdx` output helpers can own process behavior in surprising ways
- Network assumptions are embedded in help text, config names, command helpers, and endpoint resolution
- auth and browser behavior can be triggered from ordinary resource commands

Continuing to retrofit these concerns command-by-command is possible, but it will leave a hard-to-maintain hybrid where new contributors must know which helper paths are safe and which are legacy.

## Design Goals

1. Commands return data and typed errors; they do not decide process exit, JSON shape, or stderr formatting.
2. The root execution layer is the only place that converts results into stdout, stderr, and exit codes.
3. Target resolution is explicit and testable.
4. Network and self-hosted modes share the same command surface where possible.
5. OSS and OEL differences are represented as capabilities on a self-hosted target, not as separate config models.
6. Product-specific clients are injected into commands through narrow interfaces.
7. Interactive behavior is opt-in at the edge and blocked by policy under `--agent` or `--non-interactive`.
8. The machine contract is stable enough for CI systems and coding agents.

## Non-Goals

- Do not mount upstream product Cobra trees as the primary 2.0 command surface.
- Do not shell out to upstream CLI binaries.
- Do not preserve legacy command behavior when it conflicts with the agentic contract.
- Do not model OSS and OEL as separate deployment target types in the first 2.0 implementation.
- Do not use `cmdx.Print*`, `cmdx.Must`, `cmdx.Fatalf`, `cmdx.FailSilently`, or direct `os.Exit` inside user-facing command logic.

## High-Level Shape

The CLI should have four clear layers:

```text
main
  root execution
    command adapters
      services / target clients
        generated clients, product SDKs, local helpers
```

### 1. Main

`main` should be tiny:

```go
func main() {
    os.Exit(cli.Execute(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
```

No command package should call `os.Exit`.

### 2. Root Execution

The root execution package owns:

- global flags
- command registration
- context setup
- interaction policy
- output rendering
- error rendering
- exit-code mapping

It should expose an in-process API for tests:

```go
type Result struct {
    Data any
    Err  error
}

func Execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int
```

The renderer should be selected once from the root flags. Command handlers should not know whether the caller requested table output, JSON, or agent mode.

### 3. Command Adapters

Command adapters should be thin. Their job is to:

- define command names, aliases, flags, and examples
- parse flags and args into request structs
- call an application service
- return data or typed errors

They should not:

- print results
- print errors
- call `os.Exit`
- start browsers directly
- read or write config directly
- instantiate generated clients directly
- decide whether a prompt is allowed

Preferred handler shape:

```go
func runCreateProject(ctx context.Context, deps Deps, in CreateProjectInput) (*Project, error) {
    return deps.Projects.Create(ctx, in)
}
```

Cobra should be only an adapter at the edge, not the domain model.

### 4. Services And Target Clients

Services own resource workflows such as projects, identities, OAuth2 clients, relation tuples, and permissions checks.

They should depend on small interfaces:

```go
type ProjectService interface {
    List(ctx context.Context, in ListProjectsInput) ([]Project, error)
    Get(ctx context.Context, id string) (*Project, error)
    Create(ctx context.Context, in CreateProjectInput) (*Project, error)
}
```

Network and self-hosted implementations can both satisfy these interfaces when the resource exists in both modes. Unsupported operations return `unsupported_feature`.

Generated Ory API clients and upstream product libraries should live below this service boundary.

## Proposed Package Layout

This layout is intentionally explicit. It gives future agents obvious ownership boundaries.

```text
cmd/ory/
  main.go

internal/cli/
  root.go
  execute.go
  flags.go
  render.go
  errors.go
  metadata.go

internal/contract/
  envelope.go
  error_codes.go
  exit_codes.go
  schema.go

internal/interaction/
  policy.go
  prompt.go
  browser.go
  terminal.go

internal/config/
  v2.go
  load.go
  save.go
  migrate_v1.go
  profile.go
  env.go

internal/target/
  resolver.go
  target.go
  capability.go
  endpoints.go

internal/network/
  client.go
  auth.go
  projects.go
  workspaces.go

internal/selfhosted/
  clients.go
  hydra.go
  kratos.go
  keto.go
  oathkeeper.go

internal/service/
  projects.go
  identities.go
  oauth2_clients.go
  relation_tuples.go
  permissions.go

internal/commands/
  create.go
  list.go
  get.go
  update.go
  delete.go
  import.go
  validate.go
  perform.go

internal/output/
  renderer.go
  table.go
  json.go
  human.go

internal/testutil/
  execute.go
  golden.go
  fake_services.go
```

The exact names can change, but the boundaries should remain.

## Execution Contract

The Stage 1 contract from `CLI_2_0_ERROR_CONTRACT.md` should be treated as the starting point.

Global flags:

```text
--output json
--non-interactive
--agent
--no-color
```

`--agent` implies:

```text
--output json
--non-interactive
--no-color
no progress spinners
no automatic browser opening
no human success noise on stderr
```

Success envelope:

```json
{
  "schema_version": "v1",
  "data": {}
}
```

Error envelope:

```json
{
  "schema_version": "v1",
  "code": "authentication_error",
  "code_number": 1101,
  "exit_code": 4,
  "message": "authentication failed",
  "request_id": "req_...",
  "trace_id": "trace_...",
  "details": {}
}
```

The root execution layer should guarantee:

- stdout contains only command result data
- stderr contains only diagnostics or error envelopes
- JSON mode produces JSON on success and failure
- command errors are typed before rendering
- process exit happens once, in `main`

## Interaction Policy

Interactive behavior should be mediated through a single policy object:

```go
type Policy struct {
    Interactive bool
    Browser     bool
    Color       bool
    Spinner     bool
}
```

Commands and services should request interactions through interfaces:

```go
type Prompter interface {
    Confirm(ctx context.Context, prompt string) (bool, error)
    Input(ctx context.Context, prompt string) (string, error)
}

type Browser interface {
    Open(ctx context.Context, url string) error
}
```

Under `--non-interactive` or `--agent`, these return typed errors or data-only alternatives:

- prompts return `non_interactive_prompt_required`
- browser-only auth returns `authentication_error`
- `open` commands return the URL as data instead of opening the browser

No service should call `fmt.Scan`, `ReadString`, or `browser.OpenURL` directly.

## Target Model

The CLI should resolve a target before executing resource commands.

```go
type TargetKind string

const (
    TargetNetwork    TargetKind = "network"
    TargetSelfHosted TargetKind = "self-hosted"
)

type Target struct {
    Kind         TargetKind
    ProfileName  string
    Endpoints    Endpoints
    Capabilities CapabilitySet
    Auth         AuthConfig
}
```

Network profiles know about:

- console API endpoint
- selected workspace
- selected project
- Network auth/API key

Self-hosted profiles know about:

- Hydra endpoint
- Kratos endpoint
- Keto endpoint
- Oathkeeper endpoint
- auth headers or tokens
- capability set

OSS and OEL are both self-hosted. OEL-only behavior should be capability-gated:

```go
if !target.Capabilities.Has(CapabilityEnterpriseOrganizations) {
    return nil, contract.UnsupportedFeature("organizations require enterprise capabilities")
}
```

## Command Model

Primary commands should remain verb/resource oriented:

```text
ory list projects
ory get project <id>
ory create oauth2-client
ory list oauth2-clients
ory get identity <id>
ory import identities ./identities.json
ory create relation-tuple
ory check permission
```

Product names should not be required for common workflows.

Product aliases may exist as advanced compatibility or diagnostic surfaces:

```text
ory hydra ...
ory kratos ...
ory keto ...
ory oathkeeper ...
```

Those aliases should be explicitly marked as compatibility or advanced commands unless they fully satisfy the 2.0 contract.

## Upstream OSS Integration Strategy

Use adapter-first integration for the primary 2.0 command surface.

Do not mount upstream Cobra command trees directly unless a specific subtree is proven to satisfy the contract. Stage 0 validation showed that upstream command packages often mix useful client logic with direct output, fatal helpers, process exits, and human flows.

Preferred reuse order:

1. Generated clients and SDK types.
2. Product libraries that expose pure functions or client APIs.
3. Small parsing or validation helpers that do not print or exit.
4. Cobra command constructors only for explicitly isolated compatibility aliases.

When wrapping upstream functionality, the local adapter owns:

- flag names and validation
- request struct shape
- target/client selection
- output envelope
- error mapping
- interaction policy

## Config V2

Use JSON for config. It is easier for agents to inspect and safer for structured migration.

Suggested shape:

```json
{
  "version": "v2",
  "current_profile": "default",
  "profiles": {
    "default": {
      "target": "network",
      "workspace_id": "...",
      "project_id": "...",
      "auth": {
        "type": "token"
      }
    },
    "local": {
      "target": "self-hosted",
      "endpoints": {
        "hydra": "http://localhost:4445",
        "kratos": "http://localhost:4434",
        "keto": "http://localhost:4467"
      },
      "auth": {
        "type": "none"
      },
      "capabilities": ["oss"]
    }
  }
}
```

Config rules:

- preserve `ORY_CONFIG_PATH`
- support environment overrides for selected profile and endpoint URLs
- migrate `.ory-cloud.json` v1 into v2 without deleting the original unless explicitly requested
- keep profile commands data-first and testable

## Error Mapping

Use a central classifier. Product clients and service implementations should return typed errors where they know the domain. The root classifier should catch remaining errors.

Minimum stable codes:

- `unknown_error`
- `usage_error`
- `configuration_error`
- `authentication_error`
- `unsupported_feature`
- `non_interactive_prompt_required`
- `resource_not_found`
- `conflict`
- `rate_limited`
- `network_error`
- `interrupted`

Remote errors should preserve:

- HTTP status
- request ID
- trace ID
- upstream error body, where safe
- target/profile context, where useful

## Output Rendering

Use one renderer interface:

```go
type Renderer interface {
    Success(ctx context.Context, data any) error
    Error(ctx context.Context, err error) error
}
```

Human and JSON renderers can share command metadata but should not live inside command handlers.

For human table output, commands may return typed collections with optional table metadata:

```go
type TableView interface {
    Headers() []string
    Rows() [][]string
}
```

The data object remains the source of truth. Table rendering is presentation only.

Avoid adapting future command logic to `cmdx.Table` or `cmdx.TableRow`; those interfaces are useful legacy compatibility but should not define the 2.0 core model.

## Testing Strategy

Required test layers:

1. In-process root execution tests for stdout, stderr, and exit codes.
2. Golden JSON tests for success and error envelopes.
3. Unit tests for config loading, migration, profile selection, and target resolution.
4. Service tests with fake clients.
5. Contract tests that assert no user-facing command package uses:
   - `os.Exit`
   - `cmdx.Must`
   - `cmdx.Fatalf`
   - `cmdx.FailSilently`
   - direct `browser.OpenURL`
   - direct prompt reads
6. Optional browser/Playwright tests isolated behind an explicit package boundary or build tag.

Basic CLI contract tests must not require Playwright, real browsers, real Ory accounts, or live Network access.

## Migration From Current Repo

Treat the current repo as a source of components, not as the architecture.

Bring over:

- `CLI_2_0_PLAN.md`
- `CLI_2_0_STAGE_0_VALIDATION.md`
- `CLI_2_0_ERROR_CONTRACT.md`
- `CLI_2_0_PLAN_SUGGESTIONS.md`
- this architecture document
- the error-code registry concept
- JSON success/error envelope tests
- command names and aliases that still make sense
- generated clients and upstream SDK dependencies
- useful request/response mapping structs
- config migration knowledge from `.ory-cloud.json`

Rewrite or heavily refactor:

- `cmd/root.go`
- `cmd/cloudx/client.CommandHelper`
- auth flow orchestration
- output helpers
- prompt/browser handling
- target/profile resolution

Avoid bringing over unchanged:

- command handlers that print directly
- helpers that combine auth, config, target resolution, prompting, and API calls
- test packages that require browser setup for ordinary command validation
- direct upstream Cobra mounts for primary commands

## Suggested Build Sequence

### Slice 1: Skeleton And Contract

- Create the package structure.
- Implement root execution.
- Implement global flags.
- Implement success and error envelopes.
- Implement error registry and exit-code mapping.
- Add in-process execution tests.

Exit criteria:

- `ory --agent version` returns a success envelope.
- invalid flags return a JSON error envelope.
- no command code calls `os.Exit`.

### Slice 2: Config And Target Resolution

- Define config v2 structs.
- Implement load/save/migration from v1.
- Implement profile selection.
- Implement target resolver.
- Add environment overrides.

Exit criteria:

- commands can resolve `network` and `self-hosted` profiles without doing API calls.
- config tests do not touch the user's real home directory.

### Slice 3: Network Projects And Workspaces

- Implement Network auth policy.
- Implement list/get/create project and workspace services.
- Implement explicit non-interactive auth behavior.

Exit criteria:

- representative Network commands satisfy the agent contract.
- API errors preserve request IDs where available.

### Slice 4: Self-Hosted Product Adapters

- Implement Hydra OAuth2 client commands.
- Implement Kratos identity commands.
- Implement Keto relation tuple and permission commands.
- Add target capability checks.

Exit criteria:

- self-hosted commands work without product names for common workflows.
- unsupported endpoints/capabilities return `unsupported_feature`.

### Slice 5: Compatibility And Migration

- Add old-to-new command mapping docs.
- Add compatibility aliases where low risk.
- Add release notes for CI users.

Exit criteria:

- high-traffic 1.x workflows have documented 2.0 equivalents.
- compatibility aliases do not weaken the agent contract.

## Open Decisions

1. Should schema pinning use `--output json@v1`, `--schema-version v1`, or another option?
2. Should large list commands support `--output ndjson`, or should that wait until a streaming command surface exists?
3. Should product aliases be included in 2.0 GA or held for a later compatibility release?
4. What should the neutral config filename be if the CLI is no longer Network-first?
5. Which OEL capabilities should be modeled first, and how should they be detected?

## Bottom Line

The current Stage 1 branch is valuable because it made the contract concrete. It should not become the final architecture by inertia.

For maintainability, 2.0 should start with a clean execution core, command adapters that return data, explicit target resolution, and isolated product clients. Legacy code should be copied only when it fits those boundaries.
