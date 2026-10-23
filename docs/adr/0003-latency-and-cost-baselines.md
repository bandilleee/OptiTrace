# ADR-0003: Separate latency protection from cost comparisons

**Status:** Accepted for local implementation and revalidation.
**Date:** 2026-10-23
**Deciders:** Integrator following independent performance review.

## Context

The blueprint requires no more than 10% p95 regression against a strong eligible baseline and separately targets 20% lower variable cost than a competent staged baseline. The initial implementation selected its latency comparator by lowest cost. Independent review of a preserved 1,100-request workload showed that this could hide a latency regression against full parallel execution, despite meeting the absolute 500 ms target.

The phrase “strongest baseline” permits different interpretations. We adopt the conservative latency interpretation explicitly, while retaining cost-first results for diagnosis. A cheaper, slower baseline cannot certify the latency guard.

## Decision

1. The latency baseline is the fastest correct eligible alternative, with no compulsory violations and p95 at most 500 ms. The candidate must meet both the absolute 500 ms ceiling and 1.1 times that baseline p95.
2. The research cost comparator is the cheapest eligible competent staged/OPA workflow. A missed 20% target remains an honest negative finding and does not become a fabricated saving.
3. Planner ranking uses only its frozen independent tuning rows: enumerate valid candidate plans, determine the fastest estimated p95, and minimize expected configured variable cost among plans inside the absolute and relative limits. Ties resolve deterministically. Held-out measurements never become tuning observations.
4. Reports distinguish latency and cost baselines and identify the planner/comparison version. Existing reports remain immutable snapshots of their earlier semantics.

## Trade-offs and limits

Under the current 10/20/30 ms source assumptions and eight-row tuning set, the strict nominal latency guard admits an all-parallel plan for both profiles. Starting optional calls spends their synthetic charges even when an early result could otherwise settle the decision. Conditional verification permits early stopping; it does not promise skipped calls when the latency constraint requires parallel starts.

This configuration may offer no cost advantage. That is useful research evidence about the chosen workload, prices and QoS requirement, not a commercial result or proof about real telecom latency distributions. Future policy obligations remain independent of planner ranking.

The local workload retains both comparator interpretations for auditability. API p95, engine-only HTTP p95 and virtual estimates remain separate. Transport instability against frozen synthetic rows produces an explicit disagreement with effective observations; it must not be hidden by retries or by redefining the expected outcome after execution.
