# Memory Model

`clawmem` has three closely related but non-identical ways of looking at persisted data:

1. raw memory
2. trust projection
3. replay projection
4. scoped run/cycle/agent context memory

This document explains the conceptual model behind those views.

## Scoped run/cycle/agent memory

For control-plane execution, `clawmem` also stores scoped memory records keyed by:

- `repo_namespace`
- `run_namespace`
- `cycle_namespace`
- `agent_namespace`

The primary carry-forward classes are:

- `prior_cycle_summaries`
- `carry_forward_risks`
- `unresolved_gaps`
- `backlog_items`
- `reviewer_notes`

`POST /api/v1/scoped-memory/context` assembles a compact object from these classes so cycle execution can fetch context in one call.

Scoped records also use a status model:

- `open`
- `resolved`
- `superseded`
- `archived`

This enables week-run carry-forward behavior (for example, unresolved gaps or risks) while preserving clean transitions as work is resolved or superseded.

Actionable status transitions are intended for:

- `unresolved_gap`
- `carry_forward_risk`
- `backlog_item`
- `reviewer_note`
- `policy_exception`

Non-actionable classes (for example `working_context`) are stored as continuity context and are not expected to use governance transition workflows.

## Provenance model

Scoped records can carry provenance links:

- `source_run_id`
- `source_cycle_id`
- `source_artifact_id`
- `source_policy_decision_id`
- `source_model_profile_id`

These links preserve attribution across snapshot creation and run/snapshot exports.

## Snapshot references

Scoped notes can create snapshot checkpoints. A snapshot stores:

- namespace scope
- summary and creator
- referenced record ids and/or query criteria
- optional manifest references

Snapshots are exported by id and can be attached to run/cycle artifacts by the control plane.

Snapshot manifests include:

- `manifest_checksum` (sha256 hash of the manifest payload)
- `previous_snapshot_checksum` (when a prior snapshot exists in the same scope)

This provides an auditable chain suitable for governance and review workflows.

## Raw memory

Raw memory is the broad storage substrate exposed through `GET /api/v1/memories`.

It contains typed `MemoryRecord` rows with namespace and lifecycle fields such as:

- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`
- `scenario_id`
- `source_id`
- `source_ref`
- `importance`
- `pinned`
- `replay_linked`
- `retention_policy`

Raw memory is intentionally generic.
It can hold replay cases, trust artifacts, benchmark notes, scenario summaries, and future project-specific record types.

## Trust projection

The trust projection is exposed through `GET /api/v1/trust`.

Conceptually, trust records represent **scenario-stable or control-stable artifacts**:

- approvals
- mandates
- policy artifacts
- control summaries
- other trust-relevant records with a stable source identity

Trust is not best understood as an append-only event stream.
Instead, operators should treat trust artifacts as canonical records identified by scenario and source identity.

That means:

- repeated trust writes may refer to the same logical artifact lineage
- trust counts should be interpreted together with `scenario_id`, `source_id`, and `source_ref`
- a larger raw memory total does not necessarily mean the trust projection has gained new distinct logical artifacts

For DRQ, this is desirable because the trust layer is meant to represent the stable control context around a scenario, not just every repeated write.

## Replay projection

The replay projection is exposed through `GET /api/v1/replay`.

Conceptually, replay records represent **historical evidence**:

- a replay case
- a replay outcome
- an archived regression reference
- a preserved benchmark finding that should remain recallable later

Replay is central for DRQ and other adversarial regression systems because it preserves historical evidence that can be re-run or re-inspected.

Replay should therefore be thought of as:

- historical
- evidence-oriented
- useful for regression and audit
- not identical to generic memory growth

## Stable artifact vs append-oriented event behavior

`clawmem` intentionally supports both patterns:

- trust artifacts behave more like stable or canonical artifacts
- replay cases behave more like historical evidence records
- generic memory can hold either style depending on the caller

This is why consumers should not assume that all projections behave like a monotonic append log.

## Selective replay promotion into long-lived generic memory

Today, `clawmem` does not expose a clearly documented generic replay-promotion API that says:

- this replay-domain event must become a durable generic memory row
- this promoted replay case should be preserved across all future workflows

Instead, current replay persistence is application- and policy-sensitive.

In the current DRQ workflow:

- Trust Lab owns replay promotion decisions
- Trust Lab decides which replay findings matter operationally
- `clawmem` stores replay-oriented records when that workflow writes them

That means not every replay-domain event in a downstream app is guaranteed to appear as a new long-lived generic memory row.

## Why counts can differ across views

Counts can diverge legitimately across:

- `GET /api/v1/memories`
- `GET /api/v1/trust`
- `GET /api/v1/replay`
- namespace and Clawbot summary views

Reasons include:

- trust artifacts may represent stable source identity rather than distinct new evidence
- replay persistence may be selective and workflow-sensitive
- maintenance and lifecycle rules operate on raw records, not on an abstract projection counter
- projection filters only include some `memory_type` values

Operators and analytics consumers should therefore inspect:

- `scenario_id`
- `source_id`
- `source_ref`
- `memory_type`
- `retention_policy`
- `replay_linked`

instead of relying only on count deltas.

## Why this design is useful

For DRQ:

- stable trust artifacts give scenarios a canonical control context
- replay records preserve historical evidence for adversarial regression
- the two layers can evolve independently without forcing every workflow into one append-only model

For future generic platform use cases:

- compliance systems can keep stable artifact context separate from historical reviews
- review systems can preserve important replay evidence without overloading the trust projection
- regression systems can treat replay as durable evidence rather than transient telemetry

This is also why future platform work may add a more explicit replay-promotion API without changing the core storage model already in place.
