# ADR 0002: Scoped Memory for Control-Plane Carry-Forward

## Status

Accepted - April 4, 2026

## Context

`clawbot-server` now executes `replay_run`, `agent_run`, and `week_run` flows with explicit run, cycle, and agent namespaces. The control plane needs one compact context fetch before execution, scoped writes after execution, and memory snapshot references attached to run/cycle artifacts.

## Decision

### 1. Scope hierarchy is first-class

`clawmem` stores scoped records with:

- `repo_namespace`
- `run_namespace`
- `cycle_namespace` (optional)
- `agent_namespace` (optional)

This keeps the storage generic while matching ACH week-run orchestration contracts.

### 2. Carry-forward classes are explicit

Primary classes for compact carry-forward context are:

- `prior_cycle_summaries`
- `carry_forward_risks`
- `unresolved_gaps`
- `backlog_items`
- `reviewer_notes`

`POST /api/v1/scoped-memory/context` returns one assembled object instead of requiring many control-plane calls.

### 3. Status transitions are modeled

Scoped records support:

- `open`
- `resolved`
- `superseded`
- `archived`

This enables unresolved gaps/risks/backlog/reviewer notes to transition cleanly across cycles.

### 4. Snapshot model is explicit and exportable

Snapshots include:

- `snapshot_id`
- namespace scope
- `summary`, `created_by`, timestamps
- record refs and query criteria
- optional manifest reference metadata

Endpoints support create/get/list/export so run/cycle artifacts can link immutable memory references.

Snapshot manifests now include `manifest_checksum` plus optional `previous_snapshot_checksum` to support audit-friendly chaining.

### 5. Records carry provenance for governance

Scoped records can include:

- `source_run_id`
- `source_cycle_id`
- `source_artifact_id`
- `source_policy_decision_id`
- `source_model_profile_id`

This preserves attribution across compact context retrieval and exports.
### 6. Memory remains context continuity, not scored truth

Scoped memory enriches execution context and continuity. It does not replace deterministic replay authority. Authoritative scored outputs remain in run artifacts, comparisons, and review decisions managed by the control plane.

## Consequences

Positive:

- week-run cycles can carry forward context without namespace leakage
- control-plane calls are simpler (`context` + `notes` + snapshot refs)
- run-level memory exports can be attached to evidence bundles

Tradeoffs:

- scoped memory introduces additional classes/status semantics that consumers must interpret correctly
- snapshot references require discipline: evidence should cite snapshot refs rather than mutable live records
