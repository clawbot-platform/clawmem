# Architecture

`clawmem` is the memory service for the Clawbot Platform. In Phase 4 it provides a minimal but real HTTP-backed subsystem that stores structured memory records on local disk.

## Responsibilities

- persist generic memory records
- persist replay-case summaries
- persist trust-artifact summaries
- expose a stable API for future callers, especially `clawbot-trust-lab`

## Boundaries

`clawmem` does not own control-plane APIs, benchmark orchestration, trust-lab scenario execution, or ZeroClaw behavior.

- `clawbot-server` owns the platform foundation and control plane
- `clawbot-trust-lab` owns trust-lab workflows and will call this service

## Runtime Shape

1. `cmd/clawmem` loads env config and builds the app.
2. `internal/platform/bootstrap` creates the file store and services.
3. If enabled and the store is empty, seed records are loaded from `configs/memory/seed-memory.json`.
4. `internal/http/routes` exposes versioned APIs on top of the service layer.

## Storage Model

Phase 4 uses a local file-backed JSON store under `CLAWMEM_STORAGE_PATH`.

- records live under `records/<id>.json`
- each file contains one complete `MemoryRecord`
- list operations scan the directory and apply deterministic filters

This is intentionally simple. It keeps the phase explainable, inspectable, and easy to replace later with richer storage without changing the HTTP contract too early.
