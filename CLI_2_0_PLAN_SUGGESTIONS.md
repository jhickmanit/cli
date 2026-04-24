# Ory CLI 2.0 Plan — Review Suggestions

This document is review feedback on `CLI_2_0_PLAN.md`. It is meant to be read alongside that plan and either accepted, rejected, or amended by the next agent/reviewer. Each suggestion has a severity, a concrete change, and a rationale so it can be acted on without re-deriving context.

## Summary Verdict

The plan is well-scoped for a 2.0 effort. Goals, target model (`network` + `self-hosted`), and staging (contract before surface) are right. The issues below are about reducing risk in assumptions the plan currently defers, and tightening the agent-facing contract so it doesn't drift after ship.

Status categories used below:

- **Blocking** — should be resolved before Stage 1 implementation starts.
- **Important** — should be resolved before the stage that depends on it.
- **Nice-to-have** — worth doing, not worth delaying for.

---

## Blocking

### B1. Stage 0 needs an explicit fallback branch

**Where:** `### Stage 0: Upstream Integration Validation`

**Problem:** Upstream mountability is the load-bearing assumption of Stages 4 and 5. Given the risks the plan itself lists (package-level command vars, direct `os.Exit`, direct stdout writes, process-global state), "mount directly" is optimistic for most upstream commands. Today the plan has no committed behavior if Stage 0 concludes "most commands need adapter wrappers."

**Suggested change:** Add an explicit decision rule to Stage 0's acceptance criteria:

> If fewer than [threshold, e.g. 60%] of prioritized upstream commands can be mounted directly or with trivial wrappers, Stages 4 and 5 pivot to an adapter-first strategy: the CLI calls upstream client/library APIs and reimplements thin command surfaces locally. Product aliases (`ory hydra ...`) in that world mount only the subset that passes the safety checks; everything else is resource/verb-only.

This makes the plan robust to a disappointing Stage 0 result without requiring a replan.

### B2. Decide "active profile vs. Network default" now, not later

**Where:** `## Backward Compatibility Strategy` → "Open decision" paragraph; `## Recommended Decisions So Far`.

**Problem:** This is the single biggest silent-breakage risk for existing users. Scripts written against 1.x assume Network. If a machine's active profile changes (shared machine, synced dotfiles, new install defaults), those scripts break with no obvious signal.

**Suggested change:** Promote to `Recommended Decisions`:

> Migration from v1 config creates a `default` profile pinned to `target: network` and sets it as active. Commands never infer target from anywhere but the resolver. `--profile` and endpoint env vars are the only ways to change target for a given invocation.

That preserves 1.x workflow compatibility while giving 2.0 a clean mental model.

### B3. Commit to output schema stability

**Where:** `## Agentic CLI Requirements`, and `### Stage 6`.

**Problem:** The plan mandates JSON output but does not commit to versioning it. For agents, silent JSON shape changes between CLI patch versions are the worst failure mode — worse than text changes, because parsers fail hard. Without a stability commitment, agents will pin to exact CLI versions.

**Suggested change:** Add to `## Agentic CLI Requirements`:

> 11. Every JSON success envelope and error envelope includes a `schema_version` field. Breaking changes to a schema require a new version; old versions are supported for at least one minor release. `--output json` emits the current default; `--output json@v1` (or equivalent) pins a version.

And add to Stage 6 acceptance criteria: the metadata command advertises the schema versions it supports per command.

---

## Important

### I1. Collapse the three output flags

**Where:** `## Agentic CLI Requirements`, items 5–8.

**Problem:** `--output json` + `--error-format json` + `--agent` is three ways to say overlapping things. Nobody wants JSON success and text errors, so keeping them independent invites misconfiguration.

**Suggested change:**

- `--output json` implies JSON-formatted errors. Drop `--error-format` as a public flag (keep internally if needed for tests).
- `--agent` remains as the convenience umbrella that also disables interaction, color, spinners, and browser auto-open.
- Document `--agent` as equivalent to `--output json --non-interactive --no-color` plus spinner/browser suppression.

### I2. Tighten the exit-code table

**Where:** `## Agentic CLI Requirements` exit-code table.

**Problems:**

1. Code `1` is defined as both "generic failure" and "uncategorized internal error" — pick one.
2. No code for rate-limited/throttled. Agents need this distinct from generic network (`7`) to decide retry behavior.
3. No code for timeout (could fold into `7` but worth stating).

**Suggested change:**

| Code | Meaning |
| ---: | --- |
| 0 | success |
| 1 | uncategorized error (reserve for genuine unknowns) |
| 2 | usage or validation error |
| 3 | configuration error |
| 4 | authentication or authorization error |
| 5 | resource not found |
| 6 | conflict / state mismatch |
| 7 | network or remote service error (includes timeouts) |
| 8 | unsupported feature for selected target |
| 9 | rate-limited / throttled |
| 130 | interrupted |

### I3. Include request IDs and diagnostic IDs in error envelopes

**Where:** `## Agentic CLI Requirements` (new item) and error type design in Stage 1.

**Problem:** Ory APIs return request/trace IDs. When an agent fails, those IDs are the fastest path to a root cause. Current plan does not require propagating them.

**Suggested change:** Error envelope shape includes:

```json
{
  "schema_version": "v1",
  "code": "resource_not_found",
  "exit_code": 5,
  "message": "...",
  "request_id": "...",
  "trace_id": "...",
  "details": { ... }
}
```

Stage 1 should define the envelope; Stages 3–5 populate IDs from upstream responses.

### I4. Defer the capability map until a command needs it

**Where:** `## Feature Capabilities`.

**Problem:** No command in the plan branches on capabilities. Target type + "is this endpoint configured in the active profile" covers every behavior described. Introducing a capability map before it has a caller is speculative design.

**Suggested change:**

- Remove the capability list from 2.0 scope.
- Keep exit code `8` for unsupported-feature responses, driven by "endpoint not configured" or "command does not apply to this target" checks at the resolver level.
- Reintroduce a capability map when the first OEL-gated or feature-flagged command lands.

### I5. Pick a config format and stop hedging

**Where:** `## Config Model`.

**Problem:** "JSON or YAML" and "easy for humans and agents" pull different directions. Agents want JSON (native parsing, no indent ambiguity). Humans can use a dedicated edit command.

**Suggested change:**

- Commit to JSON for the on-disk format in 2.0.
- Provide `ory config edit` (opens `$EDITOR`) and `ory config set <path> <value>` for human workflows.
- Document migration from `.ory-cloud.json` to a neutral location (`$XDG_CONFIG_HOME/ory/config.json` on Linux, etc.) with `ORY_CONFIG_PATH` still honored.

### I6. Drop Oathkeeper from the 2.0 blocking path

**Where:** `### Stage 5`.

**Problem:** The plan flags Oathkeeper as the riskiest integration ("needs a dedicated integration spike"), but then makes it a gating stage for 2.0. If the spike goes poorly, 2.0 blocks.

**Suggested change:**

- Ship 2.0 without Oathkeeper; surface it behind an `--experimental` flag or a 2.1 minor.
- Keep the spike in Stage 0 scope so the decision is informed, but remove Stage 5 from the 2.0 release gate.

---

## Nice-to-have

### N1. Make stage parallelism explicit

Stage 1 (contract) blocks everything downstream, but Stages 0 (validation) and 2 (config v2) can run in parallel with Stage 1. The current numbering implies strict sequence. Add a one-line dependency graph near `## Proposed Implementation Stages`:

```text
Stage 0 ─┐
Stage 1 ─┼─▶ Stage 2 ─▶ Stage 3 ─▶ Stage 4 ─▶ Stage 6 ─▶ Stage 7
         │                          (Stage 5 optional / 2.1)
```

### N2. Resolve the `migrate` verb overlap

`ory config migrate` (config upgrade) and `ory migrate identity sql` (DB migration) use the same verb for very different operations. Not broken, but confusing in help text.

**Suggested change:** rename `ory config migrate` → `ory config upgrade`. Keep `migrate` for database operations only.

### N3. Specify the prompt-rejection error shape

Stage 1 mentions prompts are rejected with "a typed non-interactive error." Worth pinning the exact shape so the test suite and agents can match on it:

```json
{"code": "non_interactive_prompt_required", "exit_code": 2, "prompt": "..."}
```

### N4. Add a "verbose diagnostics" escape hatch

Agents sometimes need to debug their own failures. A `--diagnostics stderr` flag (or env var) that emits structured trace events to stderr — separate from the machine stdout contract — would help without polluting the clean stdout/stderr split. Could be deferred, but worth naming in the plan.

### N5. Note that JSON output must be streaming-safe

If any list command grows unboundedly (e.g., `ory list identities` against a large tenant), emitting a single JSON array holds everything in memory on both sides. Consider committing to NDJSON (`--output ndjson`) for list commands, or paginated envelopes. Worth at least an open question in the plan.

---

## Suggested Additions to "Recommended Decisions So Far"

If the above are accepted, append:

6. Migrated v1 configs create a `default` profile pinned to `network`; commands never infer target outside the resolver.
7. JSON output envelopes include `schema_version`; breaking changes bump the version and support N-1 for at least one minor.
8. `--output json` implies JSON-formatted errors; `--error-format` is not a public flag.
9. On-disk config format is JSON in 2.0; human-friendly edits go through `ory config edit` / `ory config set`.
10. Oathkeeper ships in 2.1 (or behind `--experimental` in 2.0), not as a 2.0 release gate.
11. Capability map is deferred until a concrete command requires it; exit code 8 is driven by endpoint configuration at the resolver.

---

## Open Questions Added

- Should list commands emit NDJSON by default in agent mode to stay streaming-safe on large result sets? (See N5.)
- Should `--diagnostics stderr` be a first-class flag, or rely on existing `--verbose` semantics? (See N4.)
- What is the stability promise for command metadata output (Stage 6) across minor versions?
