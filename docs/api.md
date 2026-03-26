# API

Base URL examples assume:

```text
http://127.0.0.1:8088
```

## System Endpoints

### `GET /healthz`

Returns process liveness.

### `GET /readyz`

Returns readiness based on storage availability.

### `GET /version`

Returns build metadata.

## Memory Endpoints

### `GET /api/v1/memories`

Optional query params:
- `memory_type`
- `scenario_id`

Example:

```bash
curl "http://127.0.0.1:8088/api/v1/memories?memory_type=trust_artifact"
```

### `POST /api/v1/memories`

Creates a generic memory record.

Example body:

```json
{
  "memory_type": "scenario_summary",
  "scope": "platform",
  "scenario_id": "sample-order-review",
  "source_id": "sample-pack-001",
  "summary": "Sample order review memory summary.",
  "metadata": {
    "pack_id": "sample-pack"
  },
  "tags": ["scenario", "sample"]
}
```

### `GET /api/v1/memories/{id}`

Fetches one memory record by id.

## Replay Endpoints

### `GET /api/v1/replay`

Lists replay memory records. Supports `scenario_id` filtering.

### `POST /api/v1/replay`

Stores a replay summary as a real memory record with `memory_type=replay_case`.

Example body:

```json
{
  "scenario_id": "sample-order-review",
  "source_id": "replay-case-002",
  "summary": "Replay outcome remained stable during local validation.",
  "metadata": {
    "benchmark_round_ref": "bench-local-002"
  },
  "tags": ["replay", "phase-4"]
}
```

## Trust Endpoints

### `GET /api/v1/trust`

Lists trust memory records. Supports `scenario_id` filtering.

### `POST /api/v1/trust`

Stores a trust artifact summary as a real memory record with `memory_type=trust_artifact`.

Example body:

```json
{
  "scenario_id": "sample-order-review",
  "source_id": "trust-artifact-002",
  "summary": "Mandate artifact remained internally consistent.",
  "artifact_family": "mandate",
  "artifact_type": "mandate_artifact",
  "metadata": {
    "source_flow": "local-phase-4"
  },
  "tags": ["trust", "phase-4"]
}
```

Examples in this document are illustrative only. The API is intentionally generic and can be reused by projects beyond Trust Lab or DRQ-style workflows.
