# Parallel implementation and acceptance handback

Recorded 2026-10-25. Initial revision: `0936c8bd8e2e297993e9a104224a6d4777919a24`. Final application revision under test: `925bea5568b428d984536f25c5cc79db19767533`. Later evidence-only commits do not change the tested application.

**Release verdict: HOLD.** Local functionality has substantial passing evidence, but final conditional-profile API performance and two live-browser journeys fail acceptance, real Google authentication is unverified and cloud prerequisites are incomplete. This report supersedes earlier status summaries; historical failure reports remain intact.

## Delegated work and integration

| Lane | Owned paths / inputs | Delivered evidence |
|---|---|---|
| Evaluator / algorithm | Isolated evaluator worktree; `services/engine/internal/evaluation/**`, scoped reliability/planner evidence; initial revision, then OPA repair | Scheduling-sensitive 50 ms OPA deadline reproduced under CPU throttling; bounded generated-policy event budget, cancellation/concurrency/race/fuzz evidence. Planner now minimizes tuning cost within absolute and fastest-relative p95 bounds. No changed prices, policy obligations or held-out tuning. |
| Independent acceptance | Isolated worktree; evidence/probes only; initial revision, OPA repair, then planner candidate | Real PostgreSQL role/isolation/idempotency/lease/authentication tests and directly rerunnable controller. Final review was interrupted by a usage limit; unfinished logs were preserved. Integrator reran the independent probes on final source from Linux and Windows. |
| Release preparation | Isolated worktree; `infra/**`, `.github/**`, runbooks and release evidence; initial revision | Offline Terraform/CI checks, immutable WIF identity binding, explicit staging-only release preparation, Google acceptance runbook and separate identity-only/hosted approval proposals. No cloud writes. |
| Integrator | Shared contracts, ADRs, PowerShell workload harness, targeted benchmark warning/test, integration/builds/browser validation and this evidence | Reviewed handbacks and contracts, preserved failed runs, corrected ambiguous baseline selection, rebuilt local images, measured 1,100 final authenticated evaluations and recorded unmet acceptance. User `Context/` and untracked logo/artifact kit preserved. |

Implementation commits integrated locally include evaluator `9f57e57`, planner `5036063`, release preparation `9d2613c`/`9d0d084`, measurement harness `042f886`/`4bebff2`, and presentation/contracts `925bea5`. No remote repository is configured; no PR or hosted CI success is claimed. Worktrees remain available for follow-up.

## Checks and evidence

| Check | Observed result / exit |
|---|---|
| Complete native Go/PostgreSQL suite, final planner candidate | PASS / 0; [log](acceptance-next-planner-go-tests.log) |
| Evaluator repair verification | PASS within documented scope; CPU-throttled before/after, bounded oracle, Linux race and fuzz; [report](reliability-next.md) |
| Final planner Linux race | PASS / 0, 16.076 s; [log](planner-next-linux-race.log) |
| Final independently designed HTTP probes in Linux | PASS / 0, package 4.106 s; [log](acceptance-next-planner-linux-controller.log) |
| Final native Windows HTTP plus SQL controller | PASS / 0, HTTP package 5.098 s; [log](acceptance-next-planner-windows-controller.log) |
| Web typecheck, 13 unit tests, formatting | PASS / 0; `npm run typecheck`, `npm test`, `npm run format:check`; [log](web-next-static.log) |
| Generated OpenAPI client drift check | PASS / 0, `./scripts/Generate-Client.ps1 -Check`; [log](contracts-next-drift.log) |
| Production engine and Next.js Docker builds; Compose startup | PASS / 0; actual deployed local image digests recorded in performance snapshot |
| Harness positive/negative acceptance checks | PASS / 0; catches missing rows, changed prices, bad charge totals, compulsory skips and slower cheap comparator |
| Final complete authenticated API workload | **FAIL / 1** latency, all 1,100 outcomes and charge/obligation checks pass; [report](performance-next.md), [summary](performance-next-summary.json) |
| Browser fixtures | 12/13 PASS, exit 1; isolated polling-case rerun 1/1 PASS, exit 0; [initial log](browser-next-fixtures.log), [focused log](browser-next-polling-focused.log) |
| Live browser workflows | 2/4 PASS, exit 1; evaluation persistence and benchmark agreement failed; [log](browser-next-live.log) |
| Offline Terraform validation/mock plans, workflow parsing and protocol identity tests | PASS / 0; seven mock plans; [release report](release-next.md) |
| Actual Google / hosted CI / GCP deployment, restore and rollback | NOT RUN |

The earlier journal controller failed under severe host/storage pressure before the final functional reruns passed without source or timeout changes. [Acceptance follow-up](acceptance-next-planner.md) preserves partial durable attempts, missing finalization and database wait evidence. No claim is made that memory pressure is the sole cause. Available host RAM during the final workload was approximately 137 MiB on a 15.2 GiB host; [later sample](host-next-memory.json). A stable controlled host is required to establish performance.

### Browser addendum

The fixture polling test initially failed its five-second dashboard-heading assertion. The recorded failure snapshot contains the expected heading, consistent with delayed rendering, but it does not prove when the heading appeared. The exact case passed in isolation without changing assertions or timeouts (9.3 seconds, 14.3 seconds overall). The new benchmark-disagreement warning test passed in the initial suite. This is not represented as one clean 13/13 suite run.

The live suite used real local protocol authentication, Go API, PostgreSQL, simulator and worker through Next.js, with no intercepted API routes. Policy author/reviewer approval and administrator/credential/invitation/auditor workflows passed. The evaluation journey displayed `Adapter execution failed`; the persisted [unconfirmed attempt](browser-next-unconfirmed-attempt.json) has all three starting and completed journal events, but no finalized evaluation. Its potential charges are retained; final aggregate totals remain unknown, never free or automatically replayed. The underlying journal error was not retained by the generic HTTP response, so commit acknowledgement/cancellation requires further diagnosis.

The benchmark completed and exported, but agreement acceptance correctly failed. Its new [bounded diagnostics](browser-next-benchmark-report.json) show complete-profile/full-sequential/all-clear expected `verified`, actual `indeterminate`: Number Verification was `timeout` at 256.23 ms against its 200 ms client limit; SIM/device were available. All three started calls were charged. This is a real change in fetched evidence, rather than a demonstrated OPA same-input output error. The frontend showed the new disagreement alert. No assertion was relaxed and no failed row was removed. [PostgreSQL log excerpt](browser-next-postgres.log) and ignored Playwright traces remain available locally. Both live journeys must pass on the final candidate before release.

Final full-API OptiTrace p95: combined **249.80 ms**, complete **240.04 ms**, conditional **608.12 ms**. Conditional full-parallel baseline p95: **326.40 ms**. Combined metrics cannot override either per-profile requirement. No 20% synthetic variable-cost benefit is established, and no observed MTN saving or commercial advantage is claimed.

## Remaining work in dependency order

1. Reproduce performance and failed live-browser journeys on a stable host with adequate memory; instrument journal lock/commit latency and commit acknowledgement if failures persist. Meet both 500 ms and 10% relative gates per profile without exclusions or changing the policy/price/oracle to obtain a pass. Obtain a clean full browser run.
2. Reopen/select an approved billing account. Read-only discovery on 2026-10-25 found two accounts, both closed. The user selected reopening the existing account; the console financial-account action and any payment verification remain user-operated. CLI access already exists; do not request secrets.
3. Approve the exact isolated OptiTrace identity-only project scope in [CLOUD-BOOTSTRAP](../runbooks/CLOUD-BOOTSTRAP.md), then configure Google/Identity Platform and run [real-provider acceptance](../runbooks/GOOGLE-ACCEPTANCE.md) while compute stays local.
4. Connect the repository, execute hosted CI and approve separately the immutable images, private Terraform plan, costs and hosted staging release. Complete managed PostgreSQL role/migration validation, authenticated paired acceptance, restore/rollback and notification tests. Resolve the documented stuck-queue alert/promotion-automation gaps before production.

No project, billing state, provider configuration, IAM policy or cloud resource was changed by this work. Existing services from other applications were not reused.

Cleanup: the dedicated `optitrace-acceptance-next` PostgreSQL container was stopped after the final checks; its data was retained. Start it with `docker start optitrace-acceptance-next` before rerunning the isolated controller. The main application stack remains running. User-provided `artifacts/` was not staged or modified. Tracked console logs have trailing whitespace normalized; original raw logs remain in ignored `.artifacts/` directories in the root/review worktrees. Raw workload JSON hashes were verified before committing the evidence.
