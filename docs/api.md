# API

Base URL examples assume:

```text
http://127.0.0.1:8088
```

The current API surface reflects `clawmem` V2:

- V1 generic memory, replay, and trust storage endpoints remain the foundation
- V2 adds lifecycle-aware fields plus ops endpoints for namespace or Clawbot summaries and maintenance job visibility
- V3 is not implemented yet; benchmarked lifecycle reporting and broader operator visibility remain planned

For conceptual semantics behind these endpoints, see [`memory_model.md`](./memory_model.md) and [`operations.md`](./operations.md).

## System endpoints

- `GET /healthz`
- `GET /readyz`
- `GET /version`
- `GET /metrics`

`/metrics` exposes lightweight Prometheus-style counters derived from stored records.

## Memory model

Each stored memory record includes:

- `namespace`
- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`
- `scope`
- `source_ref`
- `importance`
- `pinned`
- `replay_linked`
- `stability_score`
- `recall_count`
- `reference_count`
- `retention_policy`
- `expires_at`
- `decay_eligible_at`
- `last_accessed_at`
- `summary`
- `metadata`
- `tags`

`namespace` is derived from:

- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`

If callers omit `project_id`, `environment`, or `clawbot_id`, `clawmem` applies safe defaults for local development and generic service use.

## Memory endpoints

### `GET /api/v1/memories`

Supported query params:

- `namespace`
- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`
- `scenario_id`
- `source_ref`
- `limit`
- `offset`

The service enforces bounded page sizes. Requests above the max page size are clamped.

Example:

```bash
curl "http://127.0.0.1:8088/api/v1/memories?project_id=sample-project&memory_type=trust_artifact&limit=10&offset=0"
```

Example response:

```json
{
  "records": [
    {
      "id": "mem-1743012345000000000",
      "namespace": "sample-project/development/shared/trust_artifact",
      "project_id": "sample-project",
      "environment": "development",
      "clawbot_id": "shared",
      "memory_type": "trust_artifact",
      "scope": "trust",
      "source_id": "approval-001",
      "source_ref": "approval-001",
      "summary": "Approval artifact remained internally consistent.",
      "importance": 70,
      "pinned": false,
      "retention_policy": "standard",
      "metadata": {
        "artifact_family": "approval"
      },
      "tags": ["example", "trust"],
      "created_at": "2026-03-26T12:00:00Z",
      "updated_at": "2026-03-26T12:00:00Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0,
  "has_more": false
}
```

`/api/v1/memories` is the broad raw memory set.
It is the right place to inspect overall stored records and lifecycle fields, but it is not the same thing as the replay or trust projections.

### `POST /api/v1/memories`

Creates a generic memory record.

Supports the `Idempotency-Key` header for idempotent single-record writes. If an existing record already has the same idempotency key, the stored record is returned instead of creating a duplicate.

Example body:

```json
{
  "project_id": "sample-project",
  "environment": "development",
  "clawbot_id": "shared",
  "session_id": "session-001",
  "memory_type": "scenario_summary",
  "scope": "platform",
  "source_ref": "sample-pack-001",
  "summary": "Sample context review memory summary.",
  "importance": 60,
  "retention_policy": "standard",
  "metadata": {
    "pack_id": "sample-pack"
  },
  "tags": ["example", "scenario"]
}
```

### `POST /api/v1/memories/batch`

Creates multiple memory records in one request.

Example body:

```json
{
  "records": [
    {
      "project_id": "sample-project",
      "environment": "development",
      "clawbot_id": "shared",
      "memory_type": "benchmark_note",
      "scope": "benchmark",
      "source_ref": "note-001",
      "summary": "Stored benchmark note."
    },
    {
      "project_id": "sample-project",
      "environment": "development",
      "clawbot_id": "shared",
      "memory_type": "scenario_summary",
      "scope": "platform",
      "source_ref": "sample-pack-002",
      "summary": "Stored scenario summary."
    }
  ]
}
```

### `GET /api/v1/memories/{id}`

Fetches one memory record by id.

Reading a record also updates recall-related lifecycle fields:

- `recall_count`
- `last_accessed_at`
- `stability_score`

## Replay endpoints

### `GET /api/v1/replay`

Lists replay-oriented records. Supports `scenario_id` filtering.

This endpoint is a replay projection over stored records, not a promise that every replay-domain event in a downstream system was promoted into generic long-lived memory.

### `POST /api/v1/replay`

Stores a replay summary as a memory record with `memory_type=replay_case`.
This is a direct replay write API, not a generic replay-promotion policy API.

Example body:

```json
{
  "project_id": "sample-project",
  "environment": "development",
  "clawbot_id": "shared",
  "scenario_id": "sample-context-review",
  "source_ref": "replay-case-002",
  "summary": "Replay outcome remained stable during local validation.",
  "importance": 75,
  "retention_policy": "replay_preserve",
  "metadata": {
    "round_ref": "round-local-002"
  },
  "tags": ["example", "replay"]
}
```

Successful replay writes should not be interpreted as proof that all replay-domain activity in a caller grows monotonically in every projection.
Current replay persistence into long-lived generic memory is application- and policy-sensitive.

## Trust endpoints

### `GET /api/v1/trust`

Lists trust-oriented records. Supports `scenario_id` filtering.

This endpoint is a trust projection over stored records.
It is best interpreted as a view of stable scenario or control artifacts keyed by source identity, not as a generic append-only event counter.

### `POST /api/v1/trust`

Stores a trust or control-artifact summary as a memory record with `memory_type=trust_artifact`.

Example body:

```json
{
  "project_id": "sample-project",
  "environment": "development",
  "clawbot_id": "shared",
  "scenario_id": "sample-context-review",
  "source_ref": "trust-artifact-002",
  "summary": "Approval artifact remained internally consistent.",
  "artifact_family": "approval",
  "artifact_type": "approval_record",
  "importance": 70,
  "metadata": {
    "source_flow": "local-example"
  },
  "tags": ["example", "trust"]
}
```

Successful trust writes do not necessarily mean trust totals should be interpreted as unbounded growth in distinct logical artifacts.
Inspect `scenario_id`, `source_id`, and `source_ref` when reasoning about repeated trust writes.

## Current promotion workflow

`clawmem` does **not** currently expose a clearly documented generic replay-promotion API.

Current reality:

- `POST /api/v1/replay` stores replay-case records directly
- `POST /api/v1/trust` stores trust-artifact records directly
- downstream systems decide when and why to write those records

For the current DRQ workflow:

- Trust Lab handles replay and operator promotion decisions
- `clawmem` acts as the persistence substrate and retrieval boundary

Future platform work may expose a more explicit generic replay-promotion surface, but that does not exist as a documented `clawmem` API today.

## Error model

Errors use the shape:

```json
{
  "error": {
    "code": "bad_request",
    "message": "limit must be a valid integer",
    "status": 400
  }
}
```

Examples in this document are illustrative only. The API is intentionally generic and can be reused by projects inside or outside the Clawbot organization.

## Ops endpoints

### `GET /api/v1/ops/namespaces`

Lists namespace-level memory summaries.

Supported query params:

- `namespace`
- `project_id`
- `environment`
- `clawbot_id`
- `session_id`
- `memory_type`
- `scenario_id`
- `source_ref`

Example:

```bash
curl "http://127.0.0.1:8088/api/v1/ops/namespaces?project_id=sample-project"
```

Example response:

```json
{
  "records": [
    {
      "namespace": "sample-project/development/shared/replay_case",
      "project_id": "sample-project",
      "environment": "development",
      "clawbot_id": "shared",
      "total_records": 2,
      "records_by_type": {
        "replay_case": 2
      },
      "last_activity_at": "2026-03-26T12:00:00Z",
      "pinned_count": 0,
      "replay_linked_count": 2,
      "decay_eligible_count": 0,
      "approximate_bytes": 912,
      "average_stability": 78
    }
  ],
  "total": 1
}
```

### `GET /api/v1/ops/clawbots`

Lists Clawbot-level memory summaries using the same query filters as namespace summaries.

Example:

```bash
curl "http://127.0.0.1:8088/api/v1/ops/clawbots?project_id=sample-project"
```

### `GET /api/v1/ops/maintenance`

Returns maintenance queue counts and the latest status for each deterministic maintenance job.

Example:

```bash
curl http://127.0.0.1:8088/api/v1/ops/maintenance
```

Example response:

```json
{
  "jobs": [
    {
      "job_type": "decay_update",
      "last_result": "completed",
      "last_summary": {
        "updated": 3
      }
    },
    {
      "job_type": "expired_cleanup",
      "last_result": "",
      "last_summary": {}
    }
  ],
  "decay_queue_count": 3,
  "expired_count": 1,
  "replay_preserved_count": 4,
  "stale_summary_candidates": 2,
  "last_updated_at": "2026-03-26T12:00:00Z"
}
```

### `POST /api/v1/ops/maintenance/{job}/run`

Runs one deterministic maintenance job immediately.

Supported job ids:

- `decay_update`
- `expired_cleanup`
- `stale_summary_compaction`
- `replay_preservation_enforcement`

Example:

```bash
curl -X POST http://127.0.0.1:8088/api/v1/ops/maintenance/decay_update/run
```

Example response:

```json
{
  "job_type": "decay_update",
  "last_run_at": "2026-03-26T12:00:00Z",
  "last_duration_ms": 4,
  "last_result": "completed",
  "last_summary": {
    "updated": 3
  }
}
```
