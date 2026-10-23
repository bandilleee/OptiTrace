# Independent local acceptance — 23 October 2026

Input revision: `0936c8bd8e2e297993e9a104224a6d4777919a24`.

**Local security/recovery checks: PASS within the tested scope. Release: HOLD pending the evaluator reliability investigation, controlled performance evidence, hosted Google authentication and cloud release gates.** No new application defect was reproduced in this review. Passing short-lived processes does not clear the reported long-running evaluator anomaly.

## Isolation and method

The reviewer worked in an isolated checkout and changed only this evidence package. Existing tests and Go source were inspected, then executed against a new disposable PostgreSQL18 container `optitrace-acceptance-next`, host port55433, pinned image `postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336`. Tests use an unprivileged application login granted `optitrace_app`; migrations and fixture fault injection use the separate owner. No main-stack requests, restarts, migrations, cloud writes or real customer data were used.

Independent API probes use Go's overlay mechanism to replace the integration test only at compilation time. The original checkout's application and test files remain unchanged. The archived probe includes existing workflow setup plus additional checks. SQL probes execute in a rolled-back transaction. Sessions and billable calls are real local protocol/HTTP executions against test providers, not browser route fixtures.

## Fresh evidence

| Boundary | Evidence and result |
|---|---|
| Complete Go suite | `go test -count=1 -v ./...`, exit0. PostgreSQL tests executed, not skipped. Includes bounded exhaustive policy/stopping checks, all strategies/scenarios, adverse joint distributions, source failure classification, cancellation charging, journal failure paths, role hardening and authenticated workflow. |
| Pure human roles |35 GET authorization combinations across five single roles and seven resource groups match the contract. Tenant administrator has no implied evidence/policy access. |
| Current memberships |A signed-in author demoted to author-only immediately loses evaluation creation. Holding author+reviewer together still cannot approve one's submitted policy:409. Last administrator demotion:409. |
| Tenant isolation |Valid other-tenant member receives403 for demo scope and404 for a demo trace ID requested under their authorized tenant. Non-owner SQL reads reveal zero demo rows under other context in all seven RLS tables; forged tenant INSERT is rejected. |
| Sessions and identity |Expired session401, logout then401, disabled provider account401 after five-minute cache expiry, provider lookup503 produces401 once cache expires. Invalid callback state401, valid subsequent callback succeeds, replay401, uninvited protocol identity403. Existing identity tests reject wrong signed issuer/audience/expiry, nonce, PKCE and replay, plus production mock configuration. |
| Keys |Read-only credential cannot write, cross tenants, or access policies; revocation immediately returns401. Only hashes are persisted by inspected implementation. |
| Idempotency |First completion201; same key with omitted then explicit default strategy200 and the same trace ID; altered scenario409. Exactly one executed audit event. Concurrent duplicate during a slow execution409; final replay200; exactly three source calls total for a complete policy. Abandoned reservation and caller-disconnect durability tests also pass. |
| Jobs |An unexpired claimed job cannot be claimed twice. Expired lease is reclaimed; old-token completion cannot finalize it. Valid replacement finalizes; duplicate completion cannot overwrite the result. Expired third attempt becomes failed. Fresh server integration enqueues, executes and exports a durable benchmark. |
| Immutable evidence |Existing workflow rejects approved policy mutation/status downgrade/identity changes and self-approved inserts. Application role cannot UPDATE audit. |
| Static checks |`go vet ./...`, exit0. |

Commands and outcomes are in `acceptance-next-handback.json`. Raw outputs, the independent overlay source and SQL are archived beside this report. With the dedicated container running and PostgreSQL migrated/seeded by the baseline Go suite, run `./docs/evidence/acceptance-next-rerun.ps1` to reproduce the independent probes. The API overlay's final run passed in2.782s. Earlier probe-development failures were harness setup errors (unknown synthetic account and missing log directory), corrected before the archived final run; they did not expose an application defect.

## Findings and exact remediation

**P3 documentation drift — `contracts/AUTHORIZATION.md:41`.** It says the evaluation request is hashed before default-strategy normalization. Current `services/engine/internal/server/resources.go:164` defaults an omitted strategy before hashing at172. The independent omission/explicit-default replay returns200 with the same trace. Update that contract sentence to describe normalized hashing. No security behavior change is required. Shared contracts remain the integrator's responsibility.

## Coverage limits and next acceptance

The existing frontend evidence lists12 intercepted fixture browser journeys and4 real local-stack journeys. This lane inspected their coverage but did not rerun them; fixture role presentation does not prove Go authorization. This lane's fresh role checks directly exercise Go and PostgreSQL. It does not certify every screen for every role or provide a formal penetration test.

No hosted Google/Identity Platform account, live network supplier, IAM deployment, Cloud SQL backup restoration, regional capacity, controlled-load p95 or cost advantage was established. No fresh Linux race run or long-duration soak is claimed here. Worker tests inject expired leases and verify recovery/fencing with SQL; they do not kill an actual Cloud Run process. Provider cache expiry is injected through database timestamps rather than waiting five minutes. Cryptographic expiry tests use signed expired claims; session expiry is injected in PostgreSQL. All data and prices remain synthetic.

Re-audit the integrated evaluator repair on its exact commit before lifting the reliability hold. Preserve correctness, compulsory-check failure semantics and charges under cancellation/persistence failure. Controlled performance acceptance must run separately from concurrent compiler/test load.
