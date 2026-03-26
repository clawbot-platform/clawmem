# Development

## Local Run

```bash
cp .env.example .env
go run ./cmd/clawmem
```

Default address:

```text
127.0.0.1:8088
```

## Test Commands

```bash
go test ./...
go vet ./...
```

## Seed Behavior

`CLAWMEM_SEED_ON_STARTUP=true` seeds the local store only when the storage directory is empty. This makes startup predictable without overwriting developer-created records.

## Local Validation

```bash
curl http://127.0.0.1:8088/healthz
curl http://127.0.0.1:8088/readyz
curl http://127.0.0.1:8088/metrics
curl "http://127.0.0.1:8088/api/v1/memories?project_id=sample-project&limit=10&offset=0"
curl "http://127.0.0.1:8088/api/v1/ops/namespaces?project_id=sample-project"
curl "http://127.0.0.1:8088/api/v1/ops/clawbots?project_id=sample-project"
curl http://127.0.0.1:8088/api/v1/ops/maintenance
curl -X POST http://127.0.0.1:8088/api/v1/ops/maintenance/decay_update/run
```

## Storage Reset

The default store path is:

```text
./var/clawmem
```

Removing that directory resets the local data set.

## Backup and restore

- Backup by copying the directory pointed to by `CLAWMEM_STORAGE_PATH`
- Restore by copying the stored JSON files back into that directory
- Rebuild a local empty environment by removing the directory and restarting with seeding enabled

## Version status

`clawmem` is currently on V2.

That means local development should validate both:

- V1 generic memory, replay, and trust storage
- V2 namespace or Clawbot visibility plus deterministic lifecycle maintenance endpoints

V3 remains planned work. The current repo does not yet include benchmark-style lifecycle reports or a built-in operator dashboard.
