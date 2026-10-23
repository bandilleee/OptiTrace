# ADR-0002: Bound generated policy computation independently of scheduling

**Status:** Accepted for local implementation; cloud release remains gated.
**Date:** 2026-10-23
**Deciders:** Integrator review of the delegated evaluator investigation.

## Context

The prepared embedded OPA queries receive exactly three normalized integer states from versioned, generated decision tables. User-supplied Rego and runtime LLM decisions are unsupported. A fixed 50 ms finalization deadline could expire when the process was descheduled, returning `indeterminate` after valid evidence had already been collected. An aged service reproduced the symptom; an isolated fresh engine under a 0.1-CPU quota reproduced the actual `eval_cancel_error: context deadline exceeded` mechanism.

## Decision

Keep embedded OPA final evaluation for both `opa` and `optitrace`. Validate the three-state input shape/domain, enforce a fresh 4,096-event interpreter budget per call, and cancel through OPA's external cancellation mechanism when the budget is exhausted. Preserve direct-call context cancellation. After billable work starts, engine finalization continues to detach from the disconnected HTTP caller so completed evidence and charges survive, as before.

Actual policy/input/budget failures remain fail-closed, preserve the source evidence and charges, and emit structured error diagnostics. The fixed generated policies exercise at most 121 events across all 128 partial/terminal profile-state combinations in the current exhaustive tests. Budget headroom is guarded by tests and must be rechecked when generated policy logic or OPA changes.

## Options considered

| Option | Benefit | Limitation |
|---|---|---|
| Increase the wall-clock deadline | Small patch | Another scheduling pause can still change the verification outcome; an arbitrary larger number does not establish an execution bound |
| Replace runtime OPA with a precomputed outcome lookup | Finite computation | Changes the promised OPA-backed execution and comparison semantics; unnecessary for the demonstrated failure |
| Bound pure generated OPA work | Retains OPA and deterministic evidence semantics across scheduling delays | Adds tracer overhead; is not a general sandbox for arbitrary executable policy or expensive built-ins |

## Consequences and acceptance

- Supplier transport timeouts and started-call charging remain unchanged.
- The operation budget bounds the supported generated computation, not elapsed time when the OS withholds CPU. Full API p95 remains a separate measured gate.
- Both OPA-backed strategies incur the same safeguard overhead; cost and latency comparisons must remain honest, including negative findings.
- Regression evidence includes invalid inputs, forced budget exhaustion, cancellation/recovery concurrency, preserved disconnect evidence, exhaustive OPA equivalence, and CPU-quota before/after tests. See the [reliability investigation](../evidence/reliability-next.md).
- Local acceptance, full authenticated workload measurement, hosted Google verification and cloud release approval remain distinct gates.
