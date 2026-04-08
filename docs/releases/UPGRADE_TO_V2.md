# Upgrade to clawmem v2

## Who should use this

Use this guide when moving from older tag schemes to the semver-based `v2` release stream.

## What changed for operators

- release tags now follow semver (`v2.0.0`) with major alias (`v2`)
- publish workflow supports tag-triggered releases (`v*`) plus manual dispatch
- release publishing is gated by `go test ./...`

## Upgrade steps

1. Pull release image:

```bash
docker pull ghcr.io/clawbot-platform/clawmem:v2.0.0
```

2. Ensure persistent storage path is mounted:

- host path mapped to `/data/clawmem`
- writable by container user

3. Restart on v2 and run smoke checks:

```bash
curl -s http://127.0.0.1:8088/healthz | jq
curl -s http://127.0.0.1:8088/readyz | jq
curl -s http://127.0.0.1:8088/version | jq
```

4. Validate scoped-memory path used by control plane:

- `POST /api/v1/scoped-memory/context`
- `POST /api/v1/scoped-memory/notes`
- `GET /api/v1/scoped-memory/query`

## Rollback

Rollback is image-tag based. Redeploy the previous immutable `sha-...` image and re-run health/version checks.
