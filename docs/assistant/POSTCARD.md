# Plum Project Postcard (MVP Walking Skeleton)

## Current Goal
- Control plane (Go) with minimal HTTP API: heartbeat, assignments, create task, status.
- Agent (C++17) that heartbeats and fetches assignments.

## Key Interfaces
- POST /v1/nodes/heartbeat { nodeId, ip, labels? } -> { ttlSec }
- GET  /v1/assignments?nodeId=xxx -> { items: [ { instanceId, desired, artifactUrl, startCmd } ] }
- POST /v1/instances/status { instanceId, phase, exitCode, healthy, tsUnix }
- POST /v1/tasks { name, artifactUrl, startCmd, replicas{nodeId: n}, labels? }

## Invariants
- Desired vs Observed separation maintained in memory store.
- Idempotent task creation is not guaranteed yet (MVP). Revisit later.

## Next
- Agent: process execution + simple status report.
- Add demo script and example task payload.


