# Portfolio hosting preparation

Date: 2026-10-25, Africa/Johannesburg. Input: `4332231bcfdcef57ece61283a1b1ad171fa3df28`. Isolated worktree: `portfolio-hosting`.

**Hosting preparation PASS; hosted acceptance not run.** No provider project/service was created, linked, changed or deployed. User-approved target is Vercel + Render Free + Supabase DB/Auth Google, superseding the earlier GCP-primary selection; the parent integrator owns the corresponding root contract/ADR updates. Existing GCP infrastructure is preserved as an optional profile.

## Deliverables

- `infra/render.yaml`: one free Docker web service in Frankfurt, manual automatic-deploy setting, health endpoint, explicit portfolio startup and private `sync:false` project variables. No Render DB, worker, cron, disk, hosted migration or keepalive. File uses JSON-compatible YAML, validated against Render's current official schema.
- `infra/portfolio.Dockerfile`: pinned Go builder, non-root scratch runtime with CA certificates, portfolio/migrate/admin/healthcheck binaries, exec-form portfolio CMD. Render dockerCommand replaces CMD; there is no inherited API ENTRYPOINT or duplicate executable argument. Existing engine Dockerfile also compiles the new portfolio binary while its default remains `/app/api`.
- `scripts/hosting/Test-PortfolioConfig.ps1` plus its dependency-free test runner: static service boundaries and runtime origin/provider/key/session-pooler/TLS/pool-size/secret-location validation, with no network or secret output. Modern Supabase key prefixes are the selected hosting contract even though the Go adapter may retain legacy compatibility.
- `infra/supabase/bootstrap.sql` and `Initialize-PortfolioDatabase.ps1`: fresh dedicated `optitrace_login`, no elevated attributes/memberships, owner/application distinction, quoted psql variable, private stdin/environment credential transport, fail closed on pre-existing role, no migration/seed/auth-schema change. SupportsShouldProcess/WhatIf and localhost55435-only test mode are provided.
- CI parses/tests hosting scripts and builds/scans the portfolio image in addition to existing targets. No deployment workflow or token is added. Those new image checks require the parent's portfolio entry point to be integrated first.
- [PORTFOLIO-HOSTING.md](../runbooks/PORTFOLIO-HOSTING.md): provider prerequisites and exact Vercel/Render/Supabase settings, dual Google/Supabase callback configuration, Data API denial acceptance including `evaluation_events`, bounded free-tier behavior, migration/restore/rollback and real hosted acceptance gates.

## Validation

Final recorded checks are in [portfolio-hosting-checks.log](portfolio-hosting-checks.log):

| Check | Observed result |
|---|---|
| Static Blueprint/config validator | PASS |
| Synthetic runtime and unsafe configurations | PASS: valid setup plus 14 negative cases |
| Hosting PowerShell AST parsing | PASS, three scripts |
| Official Render Blueprint JSON Schema | PASS; public schema downloaded without account credentials; SHA256 recorded in log |
| actionlint 1.7.7 | PASS for CI and existing optional GCP workflow; shellcheck/pyflakes integrations disabled |
| Existing Compose config | PASS; default local service topology retained |
| Git whitespace | PASS |
| Local PostgreSQL bootstrap | PASS under a non-superuser CREATEROLE database owner; repeated initialization rejected |

The approved local database test used the existing pinned PostgreSQL18 image in a unique `optitrace-portfolio-bootstrap` container on localhost55435, limited to128MiB with16MiB shared buffers. A tiny ignored Go relay forwarded psql stdin and environment, allowing the Windows wrapper to run against the container's psql without installing software. Application password was absent from process arguments and filesystem. The resulting role had superuser/bypass/create-role/create-db/replication flags false and zero memberships. The container was removed after acceptance; no heavy Docker builds were performed.

## Provider/account assessment and limitations

At the start of this lane, Supabase CLI was installed but not authenticated, Vercel/Render CLIs were absent, no relevant API token environment variables or Git remote were present. The user subsequently confirmed all three provider accounts; parent owns CLI login/setup and any future reviewed resource approval. This lane installed no CLI and reused no existing hosted project.

The new `cmd/portfolio`, Supabase adapter, migration003, journal `evaluation_events` and root/web/proxy changes belong to the parent/parallel lanes and are absent from this lane's input. The Docker source change is deliberately dependent on that integrated runtime; this task does **not** claim a standalone image build, image scan, full application test, real Supavisor connection, provider role compatibility, migration003/Data API enforcement, Vercel proxy execution, real OAuth redirect acceptance, cold/warm hosted latency or deployment success. Those are integration/hosted gates.

Supabase Data API must be disabled/no application objects exposed in addition to explicit migration-level grants. Auth schema and unrelated public objects remain untouched. The callback query-state escape follows documented Supabase glob syntax but must be verified with the actual managed provider; broad wildcard hosts/previews are not an acceptable workaround.

Render sleep, Supabase inactivity pauses and quotas mean this profile is a synthetic personal portfolio, not a production SLA. The worker can recover durable leases after the next genuine wake; no uptime ping or paid daemon is configured. Report actual cloud spend separately, and never label synthetic supplier prices/results as observed MTN savings.

Current primary references, consulted2026-10-25, are linked in the hosting runbook: Render free-tier/Blueprint/schema/Docker docs, Supabase session connections/roles/Auth redirects/Data API/quotas, and Vercel project/environment/Hobby documentation. No paid upgrade or GCP provisioning was performed.
