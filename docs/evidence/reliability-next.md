# OPA finalization reliability — 23 October 2026

Input revision: `0936c8bd8e2e297993e9a104224a6d4777919a24`.

## Reproduction and diagnosis

Before restarting or rebuilding the existing API/worker, six authenticated synthetic all-clear evaluations were submitted to the main local stack. The API had been running since `2026-10-22T16:30:26.679374078Z`, with restart count zero. Complete/OPA returned `indeterminate` despite three available clear observations and all 6,600 synthetic supplier microrand being charged; its engine elapsed time was 139.72 ms. The other five requests returned `verified`. See [aged samples](reliability-next-aged-samples.json) and [the three retained historical failures](reliability-next-historical-failures.jsonl). These files contain synthetic observations only. No authentication cookie, token or credential is retained.

OPA v1.21.3 source inspection showed that prepared-query evaluations create a new evaluation context, storage transaction and cancellation object per invocation. The generated policy has no time-dependent builtins, external calls, user-provided executable policy, or query lifetime/TTL. A concurrent cancellation/recovery regression also verifies that canceled calls do not poison the shared prepared queries.

The actual failure mechanism was isolated without the main stack: a fresh engine in a Linux container limited to 0.1 CPU failed the first valid all-clear OPA finalization with `eval_cancel_error: context deadline exceeded`; its context error was also `context deadline exceeded`. The same 1,000-sample test passed without that quota on Windows. The old finalization code imposed a fresh **50 ms wall-clock deadline** after gathering observations, so scheduler suspension alone could change an otherwise valid decision into an evaluator failure. See [before diagnostic](reliability-next-linux-before.log).

The historical production-like failures swallowed the underlying OPA error. The isolated reproduction demonstrates and repairs this scheduling-sensitive mechanism; it cannot retrospectively prove the cause of every unlogged historical failure. No permanent prepared-query corruption or age-dependent expiration was demonstrated.

## Fix and correctness boundaries

The engine still executes embedded OPA on every OPA and OptiTrace finalization. It does not reuse source observations or replace the policy with a result cache/reference evaluator.

The scheduling-sensitive finalization timer is replaced by a per-evaluation **4,096 OPA trace-event work budget** with immediate interpreter cancellation when exceeded. Input must contain exactly three integers in the supported `0..3` state domain; output must be one of the three supported outcomes. This applies to the existing bounded generated, pure policies only. It is not a general sandbox for arbitrary Rego or expensive external builtins, which remain unsupported. New policy language, capabilities or OPA versions require renewed exhaustive/budget review.

The request cancellation semantics remain deliberate: before billable work starts, canceled requests fail; after started work, finalization detaches from the disconnected caller so its completed evidence can still be persisted. Direct policy-call cancellation is honored through a stopped-on-return `context.AfterFunc` and an invocation-local atomic OPA cancellation object. Adapter transport timeouts remain unchanged. Budget exhaustion and actual OPA failures still return `indeterminate`, retain original source states and charges, and now log the structured underlying error plus profile/context error. No customer observation payload is logged.

All 128 supported partial/terminal states across both profiles require at most **121 events**, leaving **33.9×** headroom. The existing exhaustive semantic/stopping oracle tests remain in force. Intentional budget exhaustion tests verify fail-closed behavior with all compulsory work and charges retained; invalid shape/domain and canceled/expired context tests exercise the boundary; a disconnect after the third durable source completion still produces the correct completed OPA decision.

## Verification

- Windows evaluation suite, including existing HTTP scenarios, benchmark correctness, journal failures, exhaustive checks and new regressions: exit 0. [Log](reliability-next-go-evaluation.log).
- `go vet ./internal/evaluation`: exit 0. [Log](reliability-next-vet.log).
- Same Linux 0.1 CPU regression, same simulator-free virtual observations and 1,000 alternating complete/conditional and OPA/OptiTrace samples: before exit 1 on sample zero; after exit 0, 1,000 correct decisions, 2.16 s test wall time. [After log](reliability-next-linux-after.log).
- Unrestricted Windows 1,000-sample and concurrent cancellation/recovery checks: exit 0. [Log](reliability-next-windows-after.log).

The quota test validates correct decisions under scheduler pressure, **not** a 500 ms API latency SLA. Virtual supplier latency/cost fields are simulation outputs; test wall time is separately reported. Controlled authenticated API performance, whole-application acceptance, real Google-provider validation and cloud release remain separate integration gates. No cloud resource or main running service was mutated by this lane.

## Reproduction commands

From `services/engine` in this worktree (PowerShell 7):

```powershell
go test ./internal/evaluation -count=1 -v
go vet ./internal/evaluation
$env:GOOS = 'linux'
$env:CGO_ENABLED = '0'
go test -c -o ../../docs/evidence/reliability-next.test ./internal/evaluation
```

Run the static test binary in an isolated local container using the existing engine image, mounting the worktree evidence directory read-only:

```powershell
docker run --rm --cpus=0.1 `
  --mount 'type=bind,source=C:/Users/bandi/.codex/worktrees/evaluator-reliability/OptiTrace/docs/evidence,target=/diagnostic,readonly' `
  --entrypoint /diagnostic/reliability-next.test optitrace-engine:local `
  '-test.run=^TestPolicyFinalizationUnderCPUQuota$' '-test.v=true'
```

The ignored `.test` executable is not part of the commit. The before diagnostic used the input implementation plus structured error logging and the same regression test; no increased timeout or restart was used to pass it.
