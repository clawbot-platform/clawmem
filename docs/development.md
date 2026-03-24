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

## Storage Reset

The default store path is:

```text
./var/clawmem
```

Removing that directory resets the local data set.
