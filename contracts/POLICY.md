# Frozen bounded policy contract: synthetic-v1

Freeze input: `b39abee7dbd3258fc793050be773305827dbbac0`, 2026-10-22.
This describes the existing bounded policy, not arbitrary customer rules. Shared-contract adoption requires integrator review. The machine-readable API is [openapi.json](openapi.json); [AUTHORIZATION.md](AUTHORIZATION.md) defines access.

## Identity and lifecycle

A policy ID identifies one immutable version. Version numbers increase within tenant + policy name. The only supported profiles are `complete` and `conditional`, with fixed freshness 60 seconds and recent-change window 24 hours. Changing semantics or thresholds requires a new contract/version and independent review; an active version must never silently acquire new meaning after an engine upgrade. Evaluations bind policy ID, version and profile and retain their original evidence.

| Transition | Actor | Guard |
|---|---|---|
| create -> draft | policy_author | Server assigns tenant, author, ID and version |
| draft -> submitted | policy_author | Actor is the original author |
| submitted -> approved | policy_reviewer | Actor differs from author; reviewer recorded |
| approved -> active | policy_reviewer | Actor differs from author; approved content unchanged |

No update/delete API, reverse transition or retirement operation exists in v1. `retired` remains a reserved response enum for compatibility, not an executable transition. Activation does not retire other versions: the caller selects an explicit active policy ID. Go checks roles/transitions and PostgreSQL protects approved content and independent review. Tenant administration alone grants no policy approval permission. Approved metadata and rule content are immutable except for the forward approved-to-active status change.

## Evidence and deterministic decisions

Capability order is number verification, SIM swap, device swap. All three are compulsory for `complete`; only number verification is compulsory for `conditional`. `available` plus explicit `value=false` means clear, and `available` plus `value=true` means adverse. A missing boolean is unknown, never clear. Unknown has no new wire enum: it normalizes to `unavailable`; unfetched work is `skipped`. Preserve `stale`, `timeout`, `rate_limited`, `permission_denied`, `unavailable` and `skipped` separately in evidence even where decision logic groups them.

Evaluate in this priority order:

1. Any compulsory check failing to resolve to valid clear/adverse evidence produces `indeterminate`, even if another check is adverse.
2. Otherwise any valid adverse observation produces `additional_verification_required`.
3. Otherwise all three checks must be valid and clear to produce `verified`.
4. Every remaining case produces `indeterminate`.

Thus an optional failure plus a valid adverse observation can require additional verification; an optional failure plus only clear observations cannot verify. No runtime LLM decides outcomes. Generated bounded Rego and the reference decision table must agree.

At evaluation time T, source age in [0,60s] is fresh; age >60s is stale. Future/missing source timestamps and wrong subject/scope are unavailable. Synthetic subject is `synthetic-subject`, scope `ZA:account-verification`. For SIM/device observations, event age in [0,24h] is adverse, >24h is clear, and a future event is unavailable. An asserted change without event time is unavailable; explicit false without event time means no observed change. Event time and source freshness time are different facts. Zero Go timestamps in failed/skipped checks are sentinels for absent evidence, not observed historical times.

## Stopping, cancellation and billing

Optimization changes fetching order/grouping only. A stop is safe only if every clear/adverse/unavailable completion of unresolved optional checks has the same outcome AND all compulsory work has completed or followed an explicit failure path. Every already-started parallel call settles and is accounted for. A compulsory check cannot be quietly skipped. Cancellation/deadline may mark unstarted compulsory remainder unavailable, with a recorded failure reason and no invented charge for an unstarted call.

Every started billable call is charged its configured synthetic price, including timeout, permission failure, cancellation and results unused by the final decision. No automatic source retry or cross-request evidence cache exists in v1. Supplier charges are integer microrand (1 rand = 1,000,000 microrand); runtime charges are separately labelled model estimates.

The API persists an idempotency reservation and attempt before source execution, then journals `starting` before dispatch and `completed` after return. A starting record is an uncertain call intention: a crash between journal commit and network dispatch cannot establish an observed charge. Unreconciled attempts remain `execution_unconfirmed`, `indeterminate`, with null aggregate costs/elapsed time and retained journal; null never means zero. Do not replay them automatically. A completed trace is immutable and sums every started call once; journal pairs must not be double-counted. Process-loss reconciliation with a real supplier remains outside synthetic-v1 acceptance.

## Decision examples

`C` = valid clear, `A` = valid adverse, `U` = resolved unavailable/failure, `P` = unfetched.

| Profile | Number / SIM / device | Outcome | May skip pending work? |
|---|---|---|---|
| complete | C / C / C | verified | No pending work |
| complete | A / U / C | indeterminate | No compulsory skips |
| conditional | A / P / P | additional_verification_required | Yes |
| conditional | C / A / P | additional_verification_required | Yes |
| conditional | C / U / C | indeterminate | No evidence of verification |
| conditional | U / P / P | indeterminate | Yes; compulsory failure recorded |

## Compatibility gate

Use exhaustive partial-state completion tests for both profiles, normalization boundary tests, and cancellation/journal tests. Changes to status meanings, compulsory sets, outcome priority, event windows or billing rules require integrator review and fresh versioning, not merely a passing schema check. See [architecture decisions](../docs/adr/0001-contract-freeze.md).
