# Final planner acceptance follow-up

Reviewed on 2026-10-25. Agent execution/logs are dated 2026-10-23. Input evaluator/planner commit: `467da2207e05b46e044fcdd100d5b5213999db01`, integrated as `5036063`. Parent presentation/contract commit: `925bea5`.

**Verdict: final functional probes PASS on integrator rerun; release HOLD for performance and cloud prerequisites.** The complete native Go/PostgreSQL suite passed, but the separate independent controller initially failed twice. The reviewer's session then reached its usage limit; the integrator preserved its unfinished handback and reran the independently designed probes on both Linux and Windows. These are integrator reruns, not a claim that the separate reviewer finished its final handback.

## What passed

- Complete `go test -count=1 -v ./...` on the candidate, using the dedicated PostgreSQL container on localhost:55433: exit 0. [Full log](acceptance-next-planner-go-tests.log). PostgreSQL tests ran.
- Review found the fastest latency comparator independent of the cheapest eligible staged/OPA cost comparator, unchanged tuning/prices, frozen expected outcomes, and a 20-detail diagnostic cap that does not exclude rows from aggregates.
- A separate injected-policy diagnostic passed and identified a limitation of `source_states_differ`: it compares all normalized positions, including legitimately skipped optional checks. Frozen `[1,0,0]` versus effective `[1,3,3]` can set the flag even when all fetched evidence agrees. It is not a causal transport-versus-policy classifier. [Probe](acceptance-next-planner-diagnostic.go.txt), [log](acceptance-next-planner-diagnostic.log). The shared contract now states this limitation.
- Parent Linux race checks on the integrated source passed: `go test -race ./internal/evaluation -count=1 -run 'TestPlanner|TestBenchmarkSeparates|TestBenchmarkAbsolute|TestBenchmarkDisagreement|TestAllStrategiesScenarios|TestDistinctHTTPFaults|TestJournal|TestCompletionJournal' -v`, exit 0, 16.076 seconds. [Log](planner-next-linux-race.log).

## Failures retained

1. During concurrent builds/race compilation, the controller's disconnected-caller probe exceeded its three-second polling window. The assertion said the trace was lost, but database inspection found an `execution_unconfirmed` attempt, all three durable starting records and two completed records. No finalized response was available. [Failure](acceptance-next-planner-loaded-failure.log), [attempt](acceptance-next-planner-loaded-attempt.log).
2. After compilation finished, the controller's initial conditional number-mismatch evaluation returned HTTP 502 after 8,094 ms, rather than 201. [Failure](acceptance-next-planner-quiet-failure.log). The durable attempt again retained three starting records and two completed records, an indeterminate/unconfirmed status and unknown final totals; automatic replay remained prohibited. [Attempt](acceptance-next-planner-quiet-attempt.json).

PostgreSQL logged cancellation while locking the evaluation-attempt resource tuple during concurrent journal updates. A checkpoint took 25.572 seconds, including slow writes/sync. [Database log](acceptance-next-planner-postgres.log). The parent observed approximately 240 MiB free host memory at that time, then approximately 137 MiB on 2026-10-25, on a 15.2 GiB host. These are concrete pressure symptoms, **not proof that resource pressure is the sole cause**. No timeout, acceptance assertion, price or policy obligation was weakened to obtain a pass.

## Required follow-up

On 2026-10-25, without changing source or increasing timeouts, the integrator ran the exact independent HTTP probes on integrated revision `925bea5`:

- Linux container sharing only the dedicated PostgreSQL container's network: exit 0, test package 4.106 seconds. [Log](acceptance-next-planner-linux-controller.log). The overlay maps `/workspace/services/engine/internal/server/integration_test.go` to the archived `acceptance-next-probes.go.txt`; it replaces tests only, not application source.
- Native Windows through `./docs/evidence/acceptance-next-rerun.ps1`: exit 0, HTTP package 5.098 seconds; subsequent transactional SQL/RLS/audit probes also completed. [Log](acceptance-next-planner-windows-controller.log).

Both include disconnect durability, concurrent idempotency and charge assertions, role/membership changes, keys, job leases and callback rejection. This clears the immediate functional rerun gap, not the observed susceptibility to journal deadlines during host/storage pressure. Recheck on a stable host with adequate memory and trace PostgreSQL lock/commit latency if the failure repeats. A repeat-until-pass latency run cannot clear the separate [failed final performance gate](performance-next.md).

All data is synthetic. The dedicated test database is isolated from the main localhost:5432 application database. No cloud mutation or real-provider acceptance was performed.
