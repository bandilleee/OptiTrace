# Latency-constrained planner and honest comparisons — 23 October 2026

Input worktree revision: `ec47f8fa6c4b04abf90687048017557957dd25fc` (accepted OPA reliability implementation plus scoped evidence). Only evaluation source and this evidence prefix changed.

## Problem and correction

The first integrated 1,100-request API matrix was correct and within 500 ms, but OptiTrace exceeded the fastest eligible baseline's p95 by more than 10%. The preceding planner minimized estimated cost subject only to 500 ms, and benchmark latency checks used the cheapest baseline's latency. Neither enforced the stronger relative-latency obligation. The original raw measurement is retained by the integrator; this change does not relabel or overwrite it.

All 13 valid sequential/parallel/mixed candidates are now replayed on the **unchanged, independent fixed eight-row tuning table**. The planner checks completed compulsory work and outcome agreement, computes nearest-rank empirical p95, identifies the fastest valid eligible candidate, then minimizes configured variable cost among candidates satisfying both `p95 <= 500 ms` and `p95 <= 1.1 * fastest_p95`. Ties use cost, p95, then stable plan name. Invalid tuning inputs yield no plan; engine initialization rejects a missing admissible plan.

Both production tuning profiles honestly select `number_verification+sim_swap+device_swap`: estimated p95 30 ms, configured supplier charge 6,600 microrand, estimated total variable cost 6,900 microrand. Optional calls launched in that batch remain fully charged even when their results establish an earlier decision. The cheaper but slower conditional plan is rejected. A separate early-adverse test fixture selects a safe number-only stopping plan, demonstrating that selection is not hardcoded parallel.

No prices, fixed tuning observations, policy rules, held-out generator or research target changed. The planner never consumes request-future or held-out observations. These tuning estimates model configured source latencies, not the complete authenticated API or actual cloud expenditure. Held-out/full-API latency must still be validated separately. No 20% saving is claimed or manufactured.

## Benchmark semantics and diagnostics

New manifests carry `planner_version: latency-constrained-v2`; the data/policy assumption version remains `synthetic-v1`. Prior reports retain their original version and semantics.

The existing comparison `baseline` now explicitly identifies the cheapest eligible **staged/OPA** controller for cost research. Optional `latency_baseline`, `latency_baseline_p95_ms`, and `latency_p95_limit_ms` independently identify the fastest correct, obligation-preserving baseline among full sequential, full parallel, staged and OPA. Missing/incorrect cohorts cannot become baselines. Missing/unpaired cost rows cannot produce research success. The 20% research target still requires correctness and both latency constraints.

At most the first **20 measured disagreements** are retained under optional report `disagreements`. Each includes profile, scenario, strategy, frozen expected outcome/states, actual outcome and the actual full result with check statuses/timestamps/timing/charges/stopping reason. `source_states_differ` flags a change in normalized observations versus a same-input output discrepancy; the evidence remains available for diagnosis rather than asserting a cause. The frozen-row oracle is computed before HTTP evaluation. **Every measured row remains in agreement aggregates; no retry, substitution, exclusion or relaxed assertion is introduced.**

This directly addresses the separately observed loaded-host browser report where full-sequential source calls produced an unexpected indeterminate result. The historical report did not retain per-call check diagnostics, so no precise retrospective transport cause is claimed.

Optional free-object report fields were approved by the integrator; shared contract/schema descriptions, ADR and browser presentation remain integrator-owned. Existing two comparison cohorts remain virtual estimates; measured engine-only and complete API acceptance are not conflated.

## Validation

- `go test ./internal/evaluation -count=1 -v`: exit 0, 2.405 s. Includes existing policy/strategy/fault/journal/charging regressions and all new tests. [Log](planner-next-evaluation.log).
- `go vet ./internal/evaluation`: exit 0. [Log](planner-next-vet.log).
- Targeted planner/comparison/diagnostic tests: exit 0. [Log](planner-next-targeted.log).
- Independent oracle replays all 13 candidates across both profiles and frozen tuning, all-clear, early-adverse, unavailable-input and all-27-terminal-state datasets. It enumerates all consistent completions independently of the production stopping memo and checks the deterministic constrained cost optimum.
- Comparison fixture gives a 25% synthetic cost reduction while breaching the fastest baseline's latency; research correctly fails. Exact relative boundary, absolute 500 ms, incorrect/missing baselines and missing paired rows are covered.
- Injected HTTP 504 responses verify the 20-record cap, retained timing/status/charges, unchanged frozen oracle and exact all-row disagreement aggregates. A private test-only policy-output fault verifies the same-input discrepancy flag separately.

Parent integrator owns targeted Linux race execution and the rebuilt controlled full-API matrix. No new heavy container build, OPA CPU-quota rerun, cloud mutation or main-stack restart was performed by this lane.

Targeted integration race expression:

```powershell
go test -race ./internal/evaluation -count=1 -run 'TestPlanner|TestBenchmarkSeparates|TestBenchmarkAbsolute|TestBenchmarkDisagreement|TestAllStrategiesScenarios|TestDistinctHTTPFaults|TestJournal|TestCompletionJournal'
```
