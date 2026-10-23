# Independent evaluator repair recheck — 23 October 2026

**PASS for the exact evaluator repair and local authorization/recovery recheck. Cloud release remains HOLD pending separate controlled performance, hosted Google and deployment acceptance.**

Original application input: `0936c8bd8e2e297993e9a104224a6d4777919a24`. Evaluator candidate reviewed: `13ad5fbc50f87f196b326483bd87f524fcd5aa92`, integrated by the parent at `9f57e57`. The reviewer independently cherry-picked the source into their isolated checkout as `d374e1c`, separately from evidence commits. `git diff --exit-code 13ad5fb -- services/engine/internal/evaluation` returned0: the evaluator source reviewed and tested exactly matches the candidate. No main-stack services or cloud resources were touched.

## Design assessment

The previous50ms finalization deadline allowed host scheduling pauses to turn already-valid evidence into an indeterminate evaluator failure. Replacing that wall timer with4096 invocation-local OPA trace events is appropriate **for the current generated, pure, fixed three-capability policy language**. The generated source contains no network/time-dependent builtin, arbitrary user program or unbounded user-provided collection. Both input shape/domain and output outcome are checked. Tests traverse all128 supported states and observe at most121 events, leaving33.9x budget headroom.

The budget is not a general security sandbox for arbitrary Rego: one expensive builtin may do work between trace events. Supporting custom executable policies, new capabilities/builtins, or upgrading OPA requires renewed work-bound review. The current code does not expose those features. This change also does not bound elapsed time under host suspension; measured500ms/p95 requirements remain separate release gates.

Before execution, canceled requests still fail without starting source work. During execution, adapters retain their200ms transport timeout; canceled compulsory remainder follows the existing unavailable failure path. Final policy computation deliberately detaches after started billable work so caller cancellation cannot discard already-collected valid evidence. Direct policy callers still honor cancellation through `context.AfterFunc`. The callback is stopped on return; its only concurrently modified object is OPA's thread-safe atomic cancellation handle, confirmed in the pinned OPA v1.21.3 `v1/topdown/cancel.go`. The event counter and budget are per invocation, not shared across prepared-query callers.

Exhaustion still fails closed to `indeterminate`, retains supplier charges and source observations, and logs an operational error without observation payloads. Budget rejection does not permit optional stopping to skip compulsory work or replay a supplier call. No authorization, contracts, policy approval, source reuse or application-session behavior was changed by this repair.

## Fresh independent verification

| Check | Result |
|---|---|
| `TEST_DATABASE_URL=...localhost:55433...; go test -count=1 -v ./...` | Exit0 against dedicated real PostgreSQL18. All original evaluation, journal, identity, authenticated server and role-hardening checks execute. |
| New evaluator regressions included in full suite |1,000 alternating finalizations, invalid inputs, exhausted budget retaining6600 synthetic supplier microrand,128-state headroom, caller disconnect after completed observations, concurrent prepared-query cancellation/recovery: PASS. The unrestricted1,000-sample check is not itself a CPU-quota test. |
| `./docs/evidence/acceptance-next-rerun.ps1` | Exit0. Repeats pure-role matrix, immediate membership revocation, dual-role self-approval rejection, tenant/resource isolation, session/provider failure behavior, scoped-key revocation and concurrent billable-call deduplication. |
| Same controller's SQL phase | Exit0. Repeats seven-table RLS/forged-write rejection, audit mutation denial, lease exclusion/takeover/fencing, immutable completed report and exhausted retries. |
| Candidate source comparison | Exit0, no evaluator diff from13ad5fb. |

Raw fresh outputs: [full Go/PostgreSQL suite](acceptance-next-recheck-go-tests.log), [independent API+SQL controller](acceptance-next-recheck-controller.log). The intentional budget-exhaustion regression emits one expected structured policy-error log; it is not an unexplained service failure. Tests ran while another lane built containers, so their wall durations are not controlled performance evidence. All compiler/test load in this acceptance lane ended before notifying the integrator to run final performance checks.

## Scope and remaining gates

No new blocking defect was found in the exact repair. The reproduced scheduler-sensitive failure mechanism is now independently accepted; this cannot retrospectively establish the cause of every historical failure whose underlying error was not logged. The reliability lane's before/after Linux0.1CPU evidence was reviewed, but this acceptance recheck did not rerun its quota container or Linux race detector. A long-duration hosted soak remains useful release evidence, not a claim from these short-lived tests.

The earlier P3 normalization-document finding was corrected separately by the integrator at `042f886`; it was not part of the evaluator diff or this checkout's tested source. The original [acceptance snapshot](acceptance-next.md) remains historical evidence for0936c8b; this recheck supersedes its pending-repair status only. Hosted Google authentication, controlled API p95 and baseline regression, backup/restore, IAM/cloud smoke tests and release approval remain separate gates. No simulated price or benchmark result is presented as observed MTN savings.
