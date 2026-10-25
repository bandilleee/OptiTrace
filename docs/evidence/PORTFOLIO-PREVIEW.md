# Hosted portfolio preview evidence

Recorded 2026-10-25. **Preview is live; full production acceptance remains HOLD.** This report updates the deployment and Google-login status in the historical [next-phase report](NEXT-PHASE.md). It does not clear its performance failures or replace independent acceptance.

## Approved release and deployed revisions

The user approved a private repository, Vercel Hobby console, Render Free API, a dedicated Supabase project, and an unbilled Google project for Google sign-in. No GCP compute/database resources or paid upgrades were provisioned. No unrelated application resources were repurposed.

| Component | Observed release |
|---|---|
| Console | [optitrace.vercel.app](https://optitrace.vercel.app), HTTP 200; Vercel production deployment `dpl_5akAg33ZNHD2Y6hbkW5BXw9eaDeY` |
| Frontend source | `d054f1af8e915a7a4dd532056ed82e43b12af388`; explicit `--local-config apps/web/vercel.json` needed to apply Frankfurt proxy region; response routing includes `fra1` |
| Go API | [health endpoint](https://optitrace-api.onrender.com/healthz), HTTP 200; Render service `srv-dbesa94s728c73d9h2og`, live deploy `dep-dbesmjfb9u1s73aqsrog` |
| API source | `e5d828a03885e8af17f845fd14393df0a608f90d` |
| PostgreSQL/Auth | Dedicated Supabase project `fsnoudfowrpurdxfgbal`, Frankfurt, PostgreSQL 17.11; migrations 1–3 applied |
| Google OAuth | Dedicated project `optitrace-auth-20261025`, billing disabled; external Testing status with the two approved Google accounts |
| Source repository | Private [bandilleee/OptiTrace](https://github.com/bandilleee/OptiTrace); provider deployments are manual, automatic deployment disabled/unconnected |

Deployments have immutable provider identifiers and source revisions. A published Render registry image digest was not exposed by the inspected deployment response and is **not verified**. Earlier healthy Render deploy `dep-dbesibrrjlhs738oq2o0` and Vercel deploy `dpl_6ikvsrs4viwwVvzi2DDjDnVwgWUy` are recorded rollback candidates; a deliberate hosted rollback has not been exercised.

## Hosted checks observed

| Check | Result |
|---|---|
| Real Google → Supabase PKCE → Go session | PASS for primary administrator and separate reviewer; application session provider-token field empty |
| Account switching | Fixed Google authorization to request `select_account`; actual chooser and both identities verified; native identity/server tests, vet and targeted Linux identity race check passed |
| Memberships | Primary account has tenant administrator, integration developer, auditor and policy author. Approved second account has only policy reviewer. No invitation email was sent by the application |
| Two-person policy workflow | Both complete and conditional version 1 created/submitted by primary identity, approved/activated by reviewer; stored author and reviewer IDs differ |
| Reviewer boundaries | Reviewer dashboard exposes overview/policies; direct membership and evaluation API access rejected; authenticated cross-tenant request rejected |
| Anonymous/session boundaries | Auth/me returns structured 401; login transaction cookie HttpOnly, Secure, SameSite=Lax and five-minute lifetime; mismatched callback rejected; real logout removed the primary session |
| Private simulator | Public `/authorize` returns 404; combined runtime uses private loopback adapters |
| Database transport/roles | Actual pooler connections use verify-full TLS with official Supabase root CA; application login has no superuser, BYPASSRLS, database/role creation or replication authority |
| Application data isolation | Data API schema exposure disabled; API roles have no application-table access; actual rollback-only two-tenant fixture confirmed RLS filtering; event table has FORCE RLS and no application update/delete grants |
| Policy immutability | Actual application-role mutation of an active policy rejected in a rollback-only transaction |
| Durable benchmark | Two-case seed-42 job completed and persisted; report popup opened; 20 strategy/cohort rows showed agreement 100% and compulsory violations zero |
| Persistence across releases | Policies, benchmark results and memberships survived subsequent Render deployments |

These two Google identities are controlled by the user. Distinct-principal authorization was tested; this is **not independent human review**. Provider-disabled-account behavior has protocol-test evidence, but a real hosted account was not disabled for this smoke test. Credential creation/revocation and hosted idempotency conflict/replay were not exercised in this hosted pass.

## Evaluation and trace evidence

The actual browser submitted the following against approved policies. Read-only SQL independently confirmed three completed resources, each with six durable call events (start and completion for each of three calls).

| Profile / scenario | Outcome | Calls | Synthetic supplier charge | Engine elapsed shown |
|---|---|---|---|---|
| Complete / clear | `verified` | 3, all compulsory | 6,600 microrand | 111.9 ms |
| Conditional / number mismatch | `additional_verification_required` | 3 | 6,600 microrand | 65.3 ms |
| Conditional / stale | `indeterminate` | 3; SIM observation distinct as stale | 6,600 microrand | 71.3 ms |

All three started calls were charged in these runs. This sample does not demonstrate call reduction: the selected plans fetched all three. Trace inspection displayed source timestamps, compulsory/optional status, charges and stopping reasons. The console can reopen the persisted trace from the execution list.

Trace identifiers: clear `Qnum2hkFyh5yWg8ftBQfZa1qt6NknzxrQtJbLxvTkVE`; mismatch `7mKGLDKO2d2jbw6OIEbmaiDRz2NJffyOW7tuftlU0rs`; stale `lkgy6Q4Ad5ux4sRjDwm4Qcj5Sp0F5MbzgXBpqvqUUgU`. Benchmark identifier: `kuy7sLBqAvmyWEt0B-ZC5AyBMYF23_IT--f-zLIpw6Y`.

These durations are individual engine measurements, **not full authenticated API p95 or browser latency**. The tiny benchmark is a workflow smoke test. Neither the 500 ms full-API gate, relative p95 gate nor 20% cost research target is established. Prices and supplier calls are synthetic; no actual MTN savings or cloud expenditure measurement is claimed.

## Recovery, automation and remaining acceptance

The pre-migration public-schema dump was saved privately and restored into a disposable local PostgreSQL database. The source public schema was empty at that point. This checks the backup tooling, **not recovery of populated application data, Supabase Auth or managed point-in-time recovery**. Hosted restore, intentional revision rollback, operational alert delivery and failure/recovery drills remain open.

GitHub Actions is blocked before jobs start. Run `40180702408` at documentation revision `94ffd4d` and API revision run `40180658140` both report `startup_failure`. The GitHub UI states recent account payments failed or a spending limit needs increasing. No billing setting was changed. Local test evidence and successful Render/Vercel builds do not establish hosted CI success.

The [integration report](portfolio-integration.md) records passing local Go/PostgreSQL, protocol auth, race, web, proxy, container and hosting-configuration checks. Account-chooser regression checks followed the final API change. Full controlled performance and independent browser/security acceptance must be rerun against the final implementation; the earlier full acceptance HOLD remains in force.

Render Free sleeps and can cold-start; Supabase Free may pause inactive projects. Availability and first-request latency are portfolio limits. Google sign-in remains invitation-only and restricted to configured test users while the OAuth app is in Testing. There are no paid upgrades, availability guarantees or automatic keep-awake requests in this release.

Private DPAPI credentials, database backups and raw provider output stay outside tracked source. Local screenshot proof is in ignored `.artifacts/release/screenshots/`; no credential or session values are included in this report. User `Context/` and untracked `artifacts/` are preserved.
