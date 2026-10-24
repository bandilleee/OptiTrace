# Authenticated API workload verification

Recorded 2026-10-25. **Performance acceptance: HOLD. Research 20% cost target: not established.** No cloud expenditure, telecom supplier saving or commercial advantage was measured.

## Method

`scripts/Measure-Workload.ps1` measures the actual authenticated Go API round trip: authorization, planning, synthetic HTTP adapters, durable journal/trace persistence and response transfer. The login uses the local protocol double through Next.js. Browser rendering, Next.js proxy latency and real Google latency are excluded from evaluation timings.

Each run has seed 20261023, ten repetitions, two equally weighted profiles, eleven equally weighted scenarios and five strategies: **1,100 measured requests**, 110 per profile/strategy, plus ten separately retained all-clear warm-ups. Case order is shuffled deterministically and strategy order rotates. The fixed independent planner tuning rows are unchanged; no future request or held-out states tune the plan. No rows are retried or removed. Measurements use one request at a time, not a concurrency/load-capacity test.

The harness rejects incomplete/duplicate matrices, unexpected outcomes, skipped compulsory work, unknown prices/costs, inconsistent charge totals and free started calls. Its positive/negative self-checks include the trap where a cheaper slower baseline would hide regression. Baseline semantics are explicit in [ADR-0003](../adr/0003-latency-and-cost-baselines.md): fastest eligible baseline protects latency; cheapest eligible staged/OPA controller is the separate research cost comparator. Nearest-rank empirical p95 is reported separately per profile and combined.

## Preserved results

| Run / profile | OptiTrace p95 | Fastest eligible baseline p95 | Absolute 500 ms | Relative 10% |
|---|---:|---:|---|---|
| Initial, combined | 252.89 ms | full parallel 221.92 ms | PASS | FAIL |
| Initial, complete | 223.95 ms | full parallel 223.95 ms | PASS | PASS |
| Initial, conditional | 280.03 ms | full parallel 219.73 ms | PASS | FAIL |
| Final, combined | 249.80 ms | full parallel 242.63 ms | PASS | PASS |
| Final, complete | 240.04 ms | full parallel 235.44 ms | PASS | PASS |
| Final, conditional | **608.12 ms** | full parallel 326.40 ms | **FAIL** | **FAIL** |

Both runs returned all **1,100 expected outcomes**, unique trace identifiers, complete compulsory work and consistent started-call charges: zero harness correctness/charge violations. Correct outcomes do not imply acceptable latency.

The initial run on 2026-10-23 used the repaired OPA evaluator but the preceding cost-first planner. Its original schema-v1 analysis selected the cheapest latency comparator and incorrectly appeared to pass. Independent review caught this; schema-v2 analysis of the **same immutable raw rows** shows 13.96% combined and 27.45% conditional regression. The original result was not overwritten. The harness and planner were corrected rather than weakening the requirement.

The final run on 2026-10-25 used commit `925bea5568b428d984536f25c5cc79db19767533`, the latency-constrained-v2 planner and services running for approximately 46 hours. It completed with command exit **1**, preserving all rows. Combined p99 was 1,703.68 ms; conditional p99 was 1,703.68 ms, so combined p95 must not hide the tail/profile failure. Several conditional requests took 0.6–2.1 seconds despite correct outcomes.

The integrator ran no concurrent agent builds/tests and submitted no benchmark jobs during the final API matrix. Windows had approximately 137 MiB available physical memory on a 15.2 GiB host; this is not a stable controlled performance environment. The observed failure is retained. Memory pressure is a plausible contributor, **not proven sole cause**. No restart, tail exclusion or repeat-until-pass measurement clears the gate. Repeat on a stable host and investigate journal/database scheduling if the problem persists.

## Cost findings and reproducibility

The final planner selects all-parallel work under the fixed tuning latency constraint, so its mean synthetic supplier charge is **6,600 microrand for either profile**, including optional calls already started. Final combined estimated variable cost was 3.31% higher than the cheapest eligible staged controller; complete-profile cost was 7.58% lower. No conditional staged comparator met the measured latency eligibility bound, so its research comparison is unavailable. The 20% target is not established in any profile. Runtime costs are configured estimates based on measured engine time, not an actual GCP bill.

[Compact evidence](performance-next-summary.json) retains all strategy/profile metrics, comparison flags, hardware, source revisions, actual container image references/digests, raw file locations and SHA-256 hashes. Full synthetic rows/checks/timestamps/trace IDs remain in ignored local `.artifacts/next-phase/workload-fixed.json` and `workload-final.json`; preserve those files when exporting this evidence. The compact snapshot is insufficient to independently replay every source check without the raw files.

```powershell
./scripts/performance/Test-Analysis.ps1
./scripts/Measure-Workload.ps1 -Repetitions 10 -OutputFile .artifacts/performance/workload.json
```

Production performance, parallel-load capacity, actual Google refresh latency, Cloud Run cold starts and browser latency remain unaccepted. A passed local workload would not clear the separate persistence/browser failures in the [final acceptance handback](NEXT-PHASE.md).
