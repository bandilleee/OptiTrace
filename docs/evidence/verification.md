# Integration verification — 22 October 2026

## Result

The local synthetic application is implemented and running. All application containers are healthy; the migration completes successfully. Local engineering verification passes. Cloud deployment remains **HOLD** until the separate release prerequisites below are satisfied.

This is integrator verification after the independent review in `independent-review.md`. That earlier report is preserved as a dated review snapshot; subsequent fixes and checks are recorded here rather than rewriting its independent findings.

## Observed checks

| Check | Observed result |
|---|---|
| Go formatting and vet | Pass |
| Full Go suite against disposable PostgreSQL18 | Pass, including protocol auth, CSRF/RBAC/RLS, independent approval, credential scope, worker/export, provider disable/session expiry, disconnect evidence and abandoned idempotency reservation |
| SQL login privilege hardening | Pass on PostgreSQL18: elevated membership/create privileges rejected, removed by migration helper, then accepted as a non-owning application role |
| Bounded policy/OPA equivalence | Pass over 128 profile/state combinations |
| Adversarial source execution | Pass over 270 strategy/profile/source-state cases |
| Durable source journaling | Pass: no network call when starting journal fails; completion failure retains partial charges and prevents further work |
| Fuzzing | Pass; 78,967 executions in the observed five-second run |
| Linux race detection | Full Go suite passes with real PostgreSQL, using a Linux builder container and current source |
| Web typing, formatting, production build | Pass |
| Web API/proxy unit tests | 9 pass |
| Controlled browser UI journeys | 5 pass |
| Real-stack browser journeys | 4 pass on hardened containers: login/evaluation/traces/mobile layout; worker benchmark/export; separate author/reviewer; credentials/invitations/auditor denial |
| OpenAPI client and sqlc generation | Generated successfully; tracked drift checked after repository bootstrap |
| Terraform | Initialization without cloud backend, validation, formatting and both mock-provider staging/production plans pass; no cloud resources created |
| PowerShell harness | Scripts parse; task graph and handback schema checks pass; missing DB fails the QA gate |
| Go vulnerability analysis | Exit0 with Go1.26.10 and updated dependencies: zero reachable vulnerabilities; one non-reachable module-level advisory remains in the tool output |
| npm audit | Zero reported vulnerabilities after updating Vitest to4.1.12 |
| Application image scan | Both application images pass High/Critical scans with the current downloaded Trivy database, without ignoring unfixed findings |

The first image scans identified High/Critical findings in the unnecessary general-purpose OS/tooling layers. The engine runtime now uses `scratch` with static binaries and a CA bundle. The web runtime uses pinned distroless Node22/Debian13. Runtime health checks remain functional without shells or curl.

## Performance evidence

The final local Docker smoke profile sends 100 authenticated all-clear evaluations through the Next.js proxy, alternating the two seeded policy profiles. It includes authentication, source work, durable journal/persistence and HTTP response transfer.

- p50: **83.12ms**
- p95: **128.77ms**
- p99: **131.87ms**
- Absolute local warm p95 <=500ms: **pass**

See `local-performance.json`. This is a local synthetic smoke profile under concurrent verification load. It does **not** establish a representative production SLA, cold-start behaviour, live Google latency, the relative10% regression gate, or20% cost savings. Engine benchmark reports separately label virtual estimates and capped20-row measured HTTP cohorts. Negative/no-benefit benchmark results are retained.

## Important repaired behaviours

- Login state/nonce/PKCE transactions are persisted and atomically consumed, rather than kept in one process.
- Started source work cannot disappear merely because the HTTP caller disconnects.
- Evaluation reservations commit before source work; interrupted keys cannot automatically replay potentially charged calls. Per-call starting/completed journal records survive independently. A starting record is uncertain, not proof of a charge.
- Application logins shed managed-service default elevated groups and fail startup if still privileged or table-owning. Cloud SQL compatibility still requires real-cloud acceptance.
- Harness acceptance runs in a credential-scrubbed child process; explicit disabled ADC configuration prevents accidental default credential fallback. QA/integrator acceptance requires a disposable DB.
- Report objects are unique to a lease attempt, referenced by the completed report and uploaded with generation preconditions. A retry cannot silently pair a different report with an earlier object's contents.
- Worker warning logs and alert filters use the same JSON field name.

## Release limitations and prerequisites

- No live Google/Identity Platform, Cloud SQL, Cloud Run, IAM/WIF or notification-delivery acceptance was performed. Supply an approved project/OAuth client, exact callback origin and secret configuration through the release process.
- No hosted GitHub Actions run was observed; workflows are implemented locally. The manual release workflow requires a successful CI run for its exact commit and protected-environment approval.
- Real Codex CLI orchestration was not launched end-to-end after the build agents reached the account usage limit. The local controller, ownership checks, schemas, independent acceptance commands and review checkpoints are implemented; actual account-backed harness execution remains to be exercised when capacity is available.
- The initial bootstrap used disjoint agent path ownership in the shared checkout before a first commit existed. Subsequent controller runs require a clean reviewed commit and use isolated worktrees; initial bootstrap is not presented as a completed run of that controller.
- No production backup restore/PITR or staged cloud rollback was executed. The runbook makes these acceptance prerequisites, not completed claims.
- Cloud cost inputs are deliberately unfilled rather than invented. Complete the price/provenance file and estimator before approving cloud spend.
- This release supports the bounded synthetic policy profiles, not arbitrary Rego, customer data or live network adapters. It establishes no MTN savings or real fraud outcomes.

Raw local evidence is retained in ignored `.artifacts/` logs and reports. Reproduce using the commands in the README and the pinned dependency/image definitions.
