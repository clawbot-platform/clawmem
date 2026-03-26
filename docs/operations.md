# Operations

This document explains how operators should interpret `clawmem` totals, projections, and lifecycle behavior in V2.

## Read the system through three views

When inspecting a running `clawmem` instance, separate these views:

1. raw memory via `GET /api/v1/memories`
2. trust projection via `GET /api/v1/trust`
3. replay projection via `GET /api/v1/replay`

They are related, but they are not interchangeable.

## How to interpret raw memory totals

Raw memory totals answer:

- how many stored records currently exist
- which memory types are present
- how records are distributed by namespace, project, or Clawbot
- how much lifecycle pressure exists from decay, expiration, or replay preservation

Raw totals do **not** by themselves tell you:

- how many distinct trust artifacts exist logically
- how many replay findings were promoted by a downstream system
- whether a count increase represents new evidence or the same source identity written again

## How to interpret trust and replay projections

Trust projection:

- use it to inspect scenario-stable or control-stable artifacts
- read it through `scenario_id`, `source_id`, and `source_ref`
- do not assume repeated writes mean distinct new trust artifacts

Replay projection:

- use it to inspect historical replay evidence
- read it through `scenario_id`, `source_ref`, and replay-oriented metadata
- do not assume every replay-domain event outside `clawmem` has been promoted into durable generic memory

## Why repeated trust creation may not mean new logical artifacts

Repeated trust writes may reuse the same logical source identity.
When that happens, operators should treat those rows as part of the same artifact lineage rather than as proof that multiple new trust artifacts were discovered.

This is especially important for:

- approvals
- mandate artifacts
- stable policy artifacts
- scenario-level control summaries

## Why replay visibility may differ from raw memory growth

Replay is historical and evidence-oriented.

A replay-domain event may:

- appear as a replay projection row
- be preserved more aggressively because it is replay-linked
- remain visible through downstream scenario retrieval
- not correspond one-to-one with generic long-lived memory growth across every workflow

For the current DRQ workflow, replay promotion decisions are handled in Trust Lab.
`clawmem` provides the persistence substrate, not the generic promotion policy surface.

## What operators should inspect

When behavior looks surprising, inspect:

- `source_id`
- `source_ref`
- `scenario_id`
- `memory_type`
- `retention_policy`
- `replay_linked`
- namespace and Clawbot summary totals
- restart behavior after reading the persisted store
- maintenance job status and last summaries

These fields explain more than raw totals alone.

## Restart and persistence behavior

`clawmem` is file-backed by default.
After restart:

- raw memory records are reloaded from the storage directory
- trust and replay projections are rebuilt from stored records
- namespace and Clawbot summaries are recalculated from persisted state

If projection counts differ from a naive expectation, inspect the underlying records rather than assuming a failed load.

## Maintenance and degraded-memory interpretation

V2 maintenance jobs operate on raw stored records:

- `decay_update`
- `expired_cleanup`
- `stale_summary_compaction`
- `replay_preservation_enforcement`

These jobs can affect what operators see in aggregate totals.

Examples:

- expired cleanup may reduce raw totals without changing the meaning of stable trust artifacts
- replay preservation may increase protected replay-linked totals without increasing the number of distinct replay scenarios
- stale compaction may remove duplicate raw rows while preserving the latest useful summary

Use `GET /api/v1/ops/maintenance`, `GET /api/v1/ops/namespaces`, and `GET /api/v1/ops/clawbots` together when diagnosing lifecycle behavior.
