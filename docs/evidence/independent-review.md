# Independent acceptance and security review

Date: 2026-10-22, Africa/Johannesburg. Reviewer: independent acceptance agent. Input revision: initial scaffold, no HEAD commit at review start. Files changed during review. No product edits by reviewer; this report is the sole reviewer write path.

## Assessment

Local backend acceptance passes with real PostgreSQL and a non-owner application login. Cloud release remains HOLD pending real configured identity/cloud acceptance, browser integration, hosted CI, backup restoration and controlled performance evidence. No cloud resources were accessed or mutated. Synthetic prices/results establish no observed MTN savings.

## Independently executed

- `go test ./...`: exit 0; initial invocation skipped DB integration because TEST_DATABASE_URL was unset.
- `go test ./... -count=1 -v` with disposable PostgreSQL on port 55432: exit 0. Exercised protocol-level login, membership authorization, CSRF denial, cross-tenant HTTP denial, independent approval/activation, auditor mutation denial, integration-key read, durable benchmark claim/completion/export, direct application-role RLS, approved-policy immutability, disconnected evaluation persistence, and logout.
- Algorithm tests in that run passed: exhaustive partial-state OPA/reference/stopping agreement, strategy/scenario agreement, freshness/event/scope boundaries, joint distributions, source faults, cancellation charges, no cross-request reuse, benchmark reproducibility and labels.
- `go vet ./...`: exit 0. `gofmt -l .`: exit 0, no output.
- `pwsh -NoProfile -File scripts/Test-Harness.ps1`: exit 0. This tests parser/task graph/schema checks, not actual coding-agent orchestration.
- `npm test -- --run`: exit 1 because vitest was not yet installed while frontend dependency installation continued. This is not passing frontend acceptance.

## Findings repaired during review

1. **P1: browser cancellation erased charged evaluation evidence.** Server now detaches a bounded evaluation transaction context before billable work. New regression cancels after a source starts and confirms durable idempotency response. Independently rerun: PASS. Process-crash/persistence-failure recovery remains a separate limitation.
2. **P2: concurrent admin demotions could remove every admin.** Membership updates now serialize with a tenant-specific advisory lock before counting admins. Inspected repair; no independent parallel-demotion regression run.
3. **P2: sqlc configured but unused.** Generated dbgen and generated audit access appeared during review. Latest Go run includes dbgen; root owns generation-drift CI.
4. **P2: browser tests mocked every API.** Frontend added OPTITRACE_LIVE=1 and acceptance.live.ts without API interception, exercising real local auth/proxy/persistence and approval/admin flows. Inspected; execution pending at this report revision.

## Integrator remediation and recheck

- **P1 repaired by inspection: acceptance inherited deployment credentials.** Harness now runs acceptance in a separate ProcessStartInfo child with shared environment scrub, explicit nonexistent ADC file, empty Cloud SDK configuration, and blocked metadata host. Disposable TEST_DATABASE_URL is retained only for acceptance. This is not adversarial OS isolation from credential files readable by the same Windows user.
- **P2 repaired and checked: QA could pass with DB tests skipped.** QA/integrator now pass RequireDatabase. Independent `Check.ps1 -Scope Engine -RequireDatabase` without URL exits 1 as expected; Test-Harness.ps1 exits 0 after repair. Backend unit-only tasks remain separate.
- **P1 repaired by inspection: Cloud SQL application login default elevated privileges.** Built-in SQL user had no custom database roles; migration only granted optitrace_app. [Google documents](https://docs.cloud.google.com/sql/docs/postgres/users) default cloudsqlsuperuser plus CREATEROLE/CREATEDB privileges. New storage/roles.go revokes unexpected groups and create privileges; API/worker check runtime privileges and table ownership. Complete Go/PostgreSQL rerun exits 0 after additions, but did not exercise these helpers at this report revision. Cloud SQL ALTER ROLE compatibility was flagged to backend: [Google documents](https://docs.cloud.google.com/sql/docs/postgres/create-manage-users) restrictions on NOREPLICATION; remove that setter and retain the assertion before release.
- **P2 repaired by inspection: alerts initially absent.** Root added latency/error/job-failure policies and database CPU saturation. Plan validation does not prove notification delivery.
- **P2 repaired by inspection: job failure log filter named the wrong field.** Worker logs WARN with JSON slog field msg; filter originally matched jsonPayload.message or ERROR. Root corrected the filter to jsonPayload.msg, so WARN benchmark failures are eligible.

## Inspection and limits

- Go obtains tenant authority from authenticated membership or credential scope before the tenant transaction. Client path tenancy is checked. Proxy forwards an allowlist and rejects traversal-like paths.
- Authentication uses PKCE, nonce, single-use state, verified Google identity and Identity Platform REST. Session lookups use hashed opaque secrets. Hosted mode rejects mocks. Correctly signed wrong issuer/audience/expired token and disabled-account adverse cases remain independently untested; library validation was inspected.
- Application/database policy guards require independent approval and immutable approved content. Audit privileges are insert/read only. Keys are hashed and plaintext returned once.
- Benchmarks label virtual estimates versus HTTP engine-only samples, cap measured samples at 20, and disclaim full-API latency certification. Representative test output did not establish the 20% target. Do not present engine timing as the authenticated API 500ms gate.
- GCP IAM, private simulator invocation, socket wiring, queue recovery and WIF were inspected, not exercised.
- Current application DB role has global queue-helper access; separate API/worker roles would improve least privilege. Privileged helper functions/session tables remain trusted Go-to-DB boundaries rather than protection from arbitrary compromised SQL.
- No independent Linux race test, vulnerability scan, live cloud login, full-stack browser run, controlled full-API performance run, restore exercise, or actual Codex harness run completed at this revision.

Changed path: docs/evidence/independent-review.md only. No commit created. Findings sent to owners/root; repaired findings rechecked as stated.
