# TESTING.md
## clawmem V2 Practical Test Plan

This document provides a practical, command-driven test plan for validating `clawmem` V2.

It focuses on:
- starting `clawmem`
- health/readiness/version checks
- inserting records through the real Trust Lab integration path
- retrieving memory records
- validating replay and trust views
- validating decay/stability behavior
- validating cleanup/maintenance behavior
- validating restart/persistence behavior
- capturing evidence for benchmark readiness
- interpreting raw memory, replay, and trust persistence correctly

This version uses commands and endpoints already visible in the current repository materials and local runs.

---

## Ports and service assumptions

This test plan assumes the following local ports:

- `clawmem` → `127.0.0.1:8088`
- `clawbot-server` → `127.0.0.1:8081`
- `clawbot-trust-lab` → `127.0.0.1:8090`

Adjust if your local ports differ.

---

## 1. Start the services

### 1.1 Start `clawmem`

From the `clawmem` repo root:

```bash
make run
```

If you prefer direct execution:

```bash
go run ./cmd/clawmem
```

### 1.2 Start `clawbot-server`

From the `clawbot-server` repo root:

```bash
make up
make smoke
SERVER_ADDRESS=127.0.0.1:8081 make run-server
```

### 1.3 Start `clawbot-trust-lab`

From the `clawbot-trust-lab` repo root:

```bash
go run ./cmd/trust-lab
```

---

## 2. Quality checks before functional testing

From the `clawmem` repo root:

```bash
go test ./...
go vet ./...
golangci-lint run ./...
gosec ./...
govulncheck ./...
```

Coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Expected outcome:
- tests pass
- vet passes
- lint passes
- security checks pass
- coverage report is generated

---

## 3. Health, readiness, and version checks

Validate `clawmem` system endpoints:

```bash
curl http://127.0.0.1:8088/healthz
curl http://127.0.0.1:8088/readyz
curl http://127.0.0.1:8088/version
```

Expected outcome:
- `/healthz` returns `{"status":"ok"}`
- `/readyz` returns `{"status":"ready"}`
- `/version` returns real build metadata if ldflags were wired, or fallback metadata if not yet updated

---

## 4. Inspect existing memory state before inserting test data

### 4.1 All memory records

```bash
curl http://127.0.0.1:8088/api/v1/memories | jq
```

### 4.2 Trust-oriented memory records

```bash
curl http://127.0.0.1:8088/api/v1/trust | jq
```

### 4.3 Replay-oriented memory records

```bash
curl http://127.0.0.1:8088/api/v1/replay | jq
```

Expected outcome:
- existing seed or previously-created records are visible
- trust records show trust-artifact style entries
- replay records show replay-case style entries

Save these outputs as your “before” baseline.

Important interpretation note:

- do not assume every trust or replay write will make every counter rise everywhere
- validate both the raw memory view and the replay or trust projections
- use `scenario_id`, `source_id`, and `source_ref` when deciding whether a write created new logical evidence or revisited an existing artifact lineage

---

## 5. Insert memory records through the real integration path

This plan uses `clawbot-trust-lab` to create records that are then persisted into `clawmem`.

### 5.1 Create a trust artifact

```bash
curl -X POST http://127.0.0.1:8090/api/v1/trust/artifacts \
  -H 'Content-Type: application/json' \
  --data '{"scenario_id":"starter-mandate-review"}' | jq
```

### 5.2 Create a replay case

```bash
curl -X POST http://127.0.0.1:8090/api/v1/replay/cases \
  -H 'Content-Type: application/json' \
  --data '{"scenario_id":"starter-mandate-review","outcome_summary":"Replay matched expected artifact flow","promotion_recommendation":"promote","promotion_reason":"Baseline outcome is explainable"}' | jq
```

### 5.3 Execute a scenario that should write supporting memory

```bash
curl -X POST http://127.0.0.1:8090/api/v1/scenarios/execute \
  -H 'Content-Type: application/json' \
  --data '{"scenario_id":"commerce-clean-agent-assisted-purchase"}' | jq
```

### 5.4 Optional: run one benchmark round that will create replay/reporting context

```bash
ROUND_ID=$(curl -s -X POST http://127.0.0.1:8090/api/v1/benchmark/rounds/run \
  -H 'Content-Type: application/json' \
  --data '{"scenario_family":"commerce"}' | jq -r '.data.id')
echo "$ROUND_ID"
```

Expected outcome:
- Trust Lab returns `201`/success responses
- new trust and replay records become visible in `clawmem`
- scenario/benchmark activity creates additional memory context

Interpretation note:

- a successful replay or trust-domain write means the integration path worked
- it does not automatically mean all projection counts should be treated as monotonic append counters
- for replay, focus on scenario retrieval and source identity
- for trust, repeated writes may reflect stable artifact identity rather than wholly new trust evidence

---

## 6. Retrieve and validate inserted records

### 6.1 Re-check all memory records

```bash
curl http://127.0.0.1:8088/api/v1/memories | jq
```

### 6.2 Re-check trust records

```bash
curl http://127.0.0.1:8088/api/v1/trust | jq
```

### 6.3 Re-check replay records

```bash
curl http://127.0.0.1:8088/api/v1/replay | jq
```

### 6.4 Validate trust status with memory context

```bash
curl "http://127.0.0.1:8090/api/v1/trust/status?scenario_id=starter-mandate-review" | jq
```

### 6.5 Validate replay status with memory context

```bash
curl "http://127.0.0.1:8090/api/v1/replay/status?scenario_id=starter-mandate-review" | jq
```

Expected outcome:
- `clawmem` record counts increase
- trust records include the new trust artifact
- replay records include the new replay case
- trust/replay status endpoints return `memory_status:"ok"` and include memory context

Do not treat raw count deltas alone as the source of truth.
Validate:

- the raw record exists where expected
- the trust or replay projection shows the expected scenario
- `source_ref` and `scenario_id` align with the intended write path

---

## 7. Practical retrieval checks

### 7.1 Verify scenario-specific records exist

```bash
curl http://127.0.0.1:8088/api/v1/memories | jq '.records[] | select(.scenario_id=="starter-mandate-review")'
```

### 7.2 Verify clean commerce scenario writes exist

```bash
curl http://127.0.0.1:8088/api/v1/memories | jq '.records[] | select(.scenario_id=="commerce-clean-agent-assisted-purchase")'
```

### 7.3 Verify replay-only view contains replay cases

```bash
curl http://127.0.0.1:8088/api/v1/replay | jq '.records[] | .record.memory_type'
```

### 7.4 Verify trust-only view contains trust artifacts

```bash
curl http://127.0.0.1:8088/api/v1/trust | jq '.records[] | .record.memory_type'
```

Expected outcome:
- scenario-specific filtering in `jq` shows the expected records
- replay endpoint returns replay cases
- trust endpoint returns trust artifacts

Replay-specific guidance:

- replay is historical evidence, not just transient activity
- not every replay-domain event is guaranteed to become durable generic memory under every workflow today
- for DRQ, replay promotion is still primarily coordinated by Trust Lab rather than by a generic `clawmem` replay-promotion API

---

## 8. Decay and stability validation

The repository already includes domain/service logic for:
- `ComputeStability`
- `RecallRecord`
- `IsExpired`
- `IsDecayEligible`
- `DecayRecord`
- maintenance jobs such as decay update, expired cleanup, stale compaction, and replay preservation

Use targeted test execution to validate those paths.

### 8.1 Domain-level decay/stability tests

```bash
go test ./internal/domain/memory -run 'Test.*(ComputeStability|RecallRecord|IsExpired|IsDecayEligible|DecayRecord)' -v
```

### 8.2 Ops-service maintenance and visibility tests

```bash
go test ./internal/services/ops -run 'Test.*(NamespaceSummaries|ClawbotSummaries|MaintenanceOverview|RunJob|runDecayUpdate|runExpiredCleanup|runStaleCompaction|runReplayPreservation)' -v
```

### 8.3 Memory service recall/update/delete behavior

```bash
go test ./internal/services/memory -run 'Test.*(ListAllUpdateDeleteAndRecall|CreateSeedListGetAndCount)' -v
```

Expected outcome:
- decay/stability behavior is deterministic
- recall updates stability/recency paths correctly
- expired cleanup and replay preservation paths behave as intended
- service-level update/delete/recall flows pass

---

## 9. Replay-linked and trust-linked preservation checks

### 9.1 Confirm replay records remain visible

```bash
curl http://127.0.0.1:8088/api/v1/replay | jq '.total'
```

### 9.2 Confirm trust records remain visible

```bash
curl http://127.0.0.1:8088/api/v1/trust | jq '.total'
```

### 9.3 Confirm scenario-linked memory context is still available after multiple reads

```bash
curl "http://127.0.0.1:8090/api/v1/trust/status?scenario_id=starter-mandate-review" | jq
curl "http://127.0.0.1:8090/api/v1/replay/status?scenario_id=starter-mandate-review" | jq
```

Expected outcome:
- replay and trust records remain available
- repeated reads do not corrupt data
- context loading remains consistent

---

## 10. Ops and control-plane validation

Use `clawbot-server` ops endpoints to validate that `clawmem` is visible as a healthy managed service.

### 10.1 Service status

```bash
curl http://127.0.0.1:8081/api/v1/ops/services/clawmem | jq
```

### 10.2 Put `clawmem` in maintenance mode

```bash
curl -X POST http://127.0.0.1:8081/api/v1/ops/services/clawmem/maintenance | jq
```

### 10.3 Resume `clawmem`

```bash
curl -X POST http://127.0.0.1:8081/api/v1/ops/services/clawmem/resume | jq
```

### 10.4 Check recent events

```bash
curl http://127.0.0.1:8081/api/v1/ops/events | jq
```

Expected outcome:
- `clawmem` shows as a managed service
- maintenance/resume transitions succeed
- events record those state changes

---

## 11. Error-path validation

### 11.1 Invalid service/scheduler checks on control plane

```bash
curl http://127.0.0.1:8081/api/v1/ops/services/does-not-exist | jq
curl http://127.0.0.1:8081/api/v1/ops/schedulers/does-not-exist | jq
curl -X POST http://127.0.0.1:8081/api/v1/ops/services/does-not-exist/maintenance | jq
```

### 11.2 Validate degraded memory behavior by stopping `clawmem`

Stop `clawmem`, then run:

```bash
curl "http://127.0.0.1:8090/api/v1/trust/status?scenario_id=starter-mandate-review" | jq
curl "http://127.0.0.1:8090/api/v1/replay/status?scenario_id=starter-mandate-review" | jq
```

Expected outcome:
- upstream services should return safe degraded-memory responses, not panic
- when `clawmem` is restarted, those paths should recover

---

## 12. Restart and persistence validation

### 12.1 Capture before-restart state

```bash
curl http://127.0.0.1:8088/api/v1/memories | jq '.total'
curl http://127.0.0.1:8088/api/v1/trust | jq '.total'
curl http://127.0.0.1:8088/api/v1/replay | jq '.total'
```

### 12.2 Restart `clawmem`

Stop the running process, then restart it:

```bash
make run
```

or:

```bash
go run ./cmd/clawmem
```

### 12.3 Re-check state after restart

```bash
curl http://127.0.0.1:8088/healthz
curl http://127.0.0.1:8088/readyz
curl http://127.0.0.1:8088/api/v1/memories | jq '.total'
curl http://127.0.0.1:8088/api/v1/trust | jq '.total'
curl http://127.0.0.1:8088/api/v1/replay | jq '.total'
curl "http://127.0.0.1:8090/api/v1/trust/status?scenario_id=starter-mandate-review" | jq
curl "http://127.0.0.1:8090/api/v1/replay/status?scenario_id=starter-mandate-review" | jq
```

Expected outcome:
- counts remain stable
- replay/trust records persist
- trust/replay status still finds memory context
- service returns healthy/ready after restart

---

## 13. Capacity smoke validation

This is not a full benchmark. It is a small practical smoke check.

### 13.1 Create multiple records through the integration path

Run the following loop a few times:

```bash
for i in {1..10}; do
  curl -s -X POST http://127.0.0.1:8090/api/v1/trust/artifacts \
    -H 'Content-Type: application/json' \
    --data '{"scenario_id":"starter-mandate-review"}' >/dev/null

  curl -s -X POST http://127.0.0.1:8090/api/v1/replay/cases \
    -H 'Content-Type: application/json' \
    --data '{"scenario_id":"starter-mandate-review","outcome_summary":"smoke-run-'$i'","promotion_recommendation":"review","promotion_reason":"capacity smoke"}' >/dev/null
done
```

### 13.2 Re-check totals

```bash
curl http://127.0.0.1:8088/api/v1/memories | jq '.total'
curl http://127.0.0.1:8088/api/v1/trust | jq '.total'
curl http://127.0.0.1:8088/api/v1/replay | jq '.total'
```

Expected outcome:
- totals increase predictably
- service remains healthy
- no obvious latency spikes or failures occur during the small smoke run

---

## 14. Evidence to capture

Save these outputs:

```bash
curl http://127.0.0.1:8088/version | jq
curl http://127.0.0.1:8088/api/v1/memories | jq > clawmem-memories.json
curl http://127.0.0.1:8088/api/v1/trust | jq > clawmem-trust.json
curl http://127.0.0.1:8088/api/v1/replay | jq > clawmem-replay.json
curl "http://127.0.0.1:8090/api/v1/trust/status?scenario_id=starter-mandate-review" | jq > trust-status.json
curl "http://127.0.0.1:8090/api/v1/replay/status?scenario_id=starter-mandate-review" | jq > replay-status.json
curl http://127.0.0.1:8081/api/v1/ops/services/clawmem | jq > clawmem-ops.json
curl http://127.0.0.1:8081/api/v1/ops/events | jq > clawmem-ops-events.json
```

Optional coverage evidence:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out > coverage-summary.txt
```

---

## 15. Pass criteria

`clawmem` V2 is ready if:

- quality checks pass
- health/readiness/version endpoints work
- records are visible in `/api/v1/memories`
- trust records are visible in `/api/v1/trust`
- replay records are visible in `/api/v1/replay`
- Trust Lab endpoints successfully create new records in `clawmem`
- trust/replay status endpoints show memory-backed context
- decay/stability-related unit tests pass
- `clawbot-server` can monitor and maintain the `clawmem` service
- restart preserves records and totals
- small capacity smoke loop does not destabilize the service

---

## 16. Fail criteria

V2 is not ready if any of the following happen:

- records disappear after restart
- trust/replay records do not appear after successful create calls
- trust/replay status endpoints cannot load memory context when `clawmem` is healthy
- maintenance/resume leaves `clawmem` in inconsistent state
- repeated small write loops create obvious instability
- decay/stability-related tests fail
- quality gate or security checks fail

---

## 17. Recommended execution order

1. start `clawmem`
2. start `clawbot-server`
3. start `clawbot-trust-lab`
4. run quality checks
5. verify system endpoints
6. capture baseline memory state
7. create trust artifact
8. create replay case
9. execute one scenario
10. optionally run one benchmark round
11. re-check `clawmem` memory/trust/replay endpoints
12. run decay/stability-focused tests
13. validate ops endpoints
14. restart `clawmem`
15. re-check persistence
16. run small capacity smoke loop
17. capture evidence
