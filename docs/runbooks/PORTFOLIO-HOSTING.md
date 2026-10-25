# Portfolio hosting: Vercel, Render Free and Supabase

Prepared 2026-10-25. This is the selected **synthetic portfolio/demo** topology. GCP Terraform and its release runbooks remain optional; do not run them for this profile. Configuration preparation is local. Creating projects, linking repositories, applying provider configuration, migrations and deployment require the separate concrete release approval. No existing unrelated project may be reused.

## Topology and prerequisites

- Vercel Hobby hosts `apps/web` (Next.js console and same-origin `/api` proxy).
- One Render Free Docker web service runs `/app/portfolio`: Go API, a loopback-only synthetic simulator and the durable benchmark lease worker in one process. No separate paid worker, cron, Cloud Run, object bucket, Render database or persistent disk is required.
- A dedicated Supabase Free project supplies PostgreSQL plus Supabase Auth's Google provider. Application sessions, invitations, memberships, roles and authorization remain in Go/PostgreSQL; a Supabase identity alone grants no OptiTrace access.

The user has provider accounts; account existence does not establish a connected CLI or permission to create a specific project. Approve the GitHub repository/branch, provider organizations/workspaces, dedicated Supabase project, Render service name, fixed Vercel production URL and invited Google accounts. Prefer the same nearby available region for Render and Supabase (the Blueprint proposes Frankfurt); this profile does not imply Johannesburg data residency. CLI tokens and passwords belong in private environment/credential stores, never a chat, Git, screenshots or handback logs.

The repository has no remote at the input revision. Linking/pushing a repository and a provider's first import can trigger deployment; perform those steps only in the approved release window. `autoDeployTrigger: off` disables subsequent automatic Render deploys, not the first Blueprint creation. The Blueprint lives at `infra/render.yaml`; select that custom path in Render. It is JSON-compatible YAML so PowerShell can validate it without a YAML parser dependency.

## Provider configuration

### Supabase database

Create a dedicated project and keep its database owner password outside Render/Vercel. Disable the Supabase Data API for the application project before migration; if other features require the API, expose a separate empty/dedicated API schema and no OptiTrace tables/functions. Do not expose `public` application objects. Disable automatic public-table exposure. Do not change the managed `auth` schema, internal Supabase roles or unrelated objects.

Use the **session pooler** connection from the Supabase Connect panel: IPv4, port **5432**, TLS `sslmode=verify-full`. The direct database endpoint can require IPv6; transaction pooling on 6543 is not the selected profile. Custom-role pooler usernames are qualified: `optitrace_login.<project-ref>`. The database owner's session-pooler username is separately `postgres.<project-ref>`. Percent-encode passwords in URLs; the application and owner must have different passwords.

Create the application login before migration using the provided PowerShell wrapper and `infra/supabase/bootstrap.sql`. The wrapper requires a unique 32-128-character random URL-safe application password (letters, digits, `_`, `-`), supplied as a SecureString. It sends the SQL variable privately through stdin and owner password through the psql process environment; neither is included in process arguments, saved files or success/error output. The SQL uses the psql quoted-variable literal, creates a fresh non-bypass login with no memberships, and fails if `optitrace_login` already exists. It never rotates an existing role silently.

```powershell
# Requires an approved database change, a native psql client and a privately set
# MIGRATION_DATABASE_URL using postgres.<ref>, verified TLS and session port 5432.
$OptiTraceAppPassword = Read-Host 'Dedicated random URL-safe application password' -AsSecureString
./scripts/hosting/Initialize-PortfolioDatabase.ps1 -ApplicationPassword $OptiTraceAppPassword
# The script creates the role only. It does not migrate, seed, deploy or print its password.
```

Run the reviewed Go `cmd/migrate` binary separately with owner `DATABASE_URL`, `APP_DATABASE_USER=optitrace_login` and `APP_ENV=production`, without `-seed`. It grants only `optitrace_app`, hardens the runtime login and checks its privileges. Keep owner credentials scoped to that operator process and clear them afterward; do not run migrations on Render startup. Approved migrations revoke Data API access on exactly the application objects, including `evaluation_events`, from `anon`, `authenticated`, `service_role` and `PUBLIC` where applicable. Disabling the Data API is an additional boundary, not a substitute for those grants.

Before starting the hosted API, verify all migration versions/checksums, a non-owner runtime login, tenant RLS and append-only audit/evaluation evidence. Confirm Data API roles cannot reach these application tables: `schema_migrations`, `tenants`, `users`, `memberships`, `invitations`, `sessions`, `login_transactions`, `resources`, `credentials`, `audit`, `jobs`, `idempotency`, and `evaluation_events`, or the application sequences/security-definer functions. Never use broad revokes against unrelated Supabase objects. The managed provider's role permissions and pooler behavior remain hosted acceptance requirements.

Recommended runtime connection (placeholder, not a credential):

```text
postgresql://optitrace_login.<project-ref>:<encoded-application-password>@<dashboard-session-pooler-host>:5432/postgres?sslmode=verify-full&pool_max_conns=4&pool_min_conns=0&pool_max_conn_lifetime=300s&pool_max_conn_idle_time=60s
```

### Google through Supabase Auth

Use a dedicated Google web OAuth client. Register **Google's redirect URI** as `https://<project-ref>.supabase.co/auth/v1/callback` (copy the exact provider callback from Supabase), then store the Google client ID/secret in the Supabase Google-provider configuration. Those Google credentials do not belong in Vercel or the Render application environment. Configure the required consent/test accounts; enable only the intended Google sign-in method for this application. No Firebase SDK/emulator or browser Supabase SDK is required by the Go-managed flow.

Set Supabase Auth Site URL to the fixed Vercel production origin, for example `https://<approved-app>.vercel.app`. Configure the app redirect allowlist to its exact callback path. The Go flow adds its random login state to `redirect_to`, so the allowlist must permit that query:

```text
https://<approved-app>.vercel.app/api/v1/auth/callback
https://<approved-app>.vercel.app/api/v1/auth/callback\?state=*
```

The second entry uses Supabase's documented `\c` literal escaping for `?`; its host and callback path are fixed, and only the random state value varies. Verify acceptance with the real configured provider before release; do not fall back to wildcard hosts, all Vercel previews or `/**`. The Google callback above and the application callback here are different URLs. Supabase owns upstream OAuth state; Go independently checks its own state/transaction, PKCE, Google identity, invitation and membership before issuing an opaque application cookie. Provider tokens are transient, not application browser sessions.

### Render Free

Create a new service from `infra/render.yaml` only after approval. The dedicated `infra/portfolio.Dockerfile` has exec-form `CMD ["/app/portfolio"]` and no inherited API ENTRYPOINT; the Blueprint's `dockerCommand` replaces CMD safely. The existing engine Dockerfile still defaults to `/app/api` for local Compose/optional GCP, and now also compiles the portfolio binary.

The portfolio entry point selects the portfolio runtime profile internally and creates its simulator URL on loopback; do not configure external simulator/job/bucket variables. Port 10000 is declared in the Blueprint; the runtime binds Render's `PORT` on all interfaces. The simulator must never be published as another web service or public port.

| Render variable | Value/boundary |
|---|---|
| `APP_ENV` | `production` |
| `AUTH_MODE` | `supabase` |
| `PORT` | `10000` (Blueprint; use Render's supplied port) |
| `APP_ORIGIN` | Fixed approved HTTPS Vercel production origin, no slash/path/query |
| `API_PUBLIC_URL` | That exact origin plus `/api` |
| `SUPABASE_URL` | Dedicated project HTTPS URL |
| `SUPABASE_PUBLISHABLE_KEY` | Project's modern `sb_publishable_...` key; keep server-side for this flow |
| `SUPABASE_SECRET_KEY` | Modern `sb_secret_...`, Render secret only; never `NEXT_PUBLIC_` |
| `DATABASE_URL` | Dedicated non-bypass custom-role session pooler URL above |

All project-specific Blueprint variables use `sync: false` and have no committed values. Prefer modern Supabase keys; the hosting validator intentionally requires modern key prefixes even if the Go adapter supports legacy keys for compatibility. Do not put `MIGRATION_DATABASE_URL`, `SIMULATOR_URL`, `SIMULATOR_AUDIENCE`, `BENCHMARK_JOB_NAME`, `REPORT_BUCKET` or mock-provider configuration into Render.

### Vercel

Import the approved repository as a Next.js project with **Root Directory `apps/web`**, Node.js 22, install command `npm ci`, build command `npm run build`, and standard Next.js output handling. Pin the production branch and review the first deployment before exposing access. Use one stable production domain as the identity/session origin. Set production-only, server-side `API_BASE_URL=https://<approved-service>.onrender.com` (no `/api`, no trailing path). No database URL, Supabase secret key or Google client secret belongs in Vercel or any `NEXT_PUBLIC_` variable.

Vercel's proxy forwards `/api/v1/...` to Go `/v1/...`, including the browser session cookie, CSRF header and original Origin, and returns Set-Cookie to the first-party Vercel origin. Direct browser-to-Render auth is not the application flow. Keep preview deployments on UI fixtures or a separately approved isolated backend; do not route preview authentication to production or add wildcard redirect hosts. Configure a function region near Render/Supabase when available and an appropriate proxy time budget; a sleeping free backend may outlast an individual browser/API request and must produce a clear retry state rather than an empty successful result.

## Free-tier behavior and bounds

This is a personal, non-commercial portfolio profile under the applicable provider terms, not a production availability commitment. Current primary sources were checked 2026-10-25:

- Render Free sleeps after 15 minutes without inbound traffic and takes about a minute to wake. One workspace shares 750 free instance-hours/month. No persistent disk is available. Work in process can be interrupted; durable jobs are stored in PostgreSQL and recover using existing leases on the next genuine user-driven wake. No self-ping, external keepalive or paid worker is provisioned.
- Supabase Free currently has a 500 MB database, 5 GB egress, 50,000 MAU and two active projects; projects can pause after a week without activity. A paused project needs operator attention, not an infinite API retry loop. Do not use Render Free Postgres as a replacement: its free database expires after 30 days.
- Keep synthetic benchmarks small (start at 10-40 observations), one visible test run at a time, and respect the API's hard bounds. Record database/report growth and remaining quotas before extended demonstrations. No object bucket or local persisted files are needed; durable report/trace data lives in PostgreSQL. Do not remove immutable evidence ad hoc to fit a quota; approve an isolated demonstration reset or a retention design separately.
- Cold starts, pauses, provider throttling and network distance are reported separately from warm measurements. The local 500 ms warm API target is not a hosted free-tier SLA, and no current hosted performance certificate or simulated supplier-savings claim is made. Free allowances are shared/conditional; a plan upgrade, extra resource or commercial use needs a new cost/release review.

## Validation, release and recovery

Local preparation checks (no provider mutation):

```powershell
./scripts/hosting/Test-PortfolioConfig.ps1 -Mode Static
./scripts/hosting/Test-PortfolioConfig.Tests.ps1
# With privately loaded candidate Render variables, validates shape/boundaries only:
./scripts/hosting/Test-PortfolioConfig.ps1 -Mode Runtime
# Optional official schema check; downloads public metadata, not account data:
New-Item -ItemType Directory -Path .artifacts -Force | Out-Null
Invoke-WebRequest https://render.com/schema/render.yaml.json -OutFile .artifacts/render.schema.json
Test-Json -Json (Get-Content infra/render.yaml -Raw) -SchemaFile .artifacts/render.schema.json
```

Before the first hosted release, review exact project/service IDs, repository/revision, stable origin/callbacks, modern secret keys, selected free plans, SQL bootstrap/migration evidence and rollback candidates. Creating/importing a Blueprint or a Vercel project is a deployment action even with later auto-deploy disabled. Bootstrap an initial tenant and invited real Google identities with the privileged administrative CLI in a private operator session; never seed synthetic identities into hosting. Use independent author/reviewer accounts for approval acceptance.

Authenticated hosted acceptance must cover invited/uninvited Google accounts, callback replay/state/PKCE and Google-only identity, membership removal, disabled/banned-provider account status, cookie flags/logout/session expiry, CSRF, all five human roles and their unions, cross-tenant denial, API key revocation, evaluation/charged-call evidence, trace export, policy self-approval rejection, benchmark polling, interrupted lease recovery and the sleep/wake retry experience. Confirm the simulator and mock identity endpoints are not publicly reachable. Test both provider pauses and clean restarts without manufacturing keepalive traffic.

Take a private logical database export before migrations, verify restoration to a separately approved disposable database, and record schema version/row counts/tenant-isolation evidence. Do not assume a free managed PITR/restore facility. Record previous Vercel and Render deployment IDs plus image/source revision; Render Free retains only limited previous deploys. Roll back application deployments only to schema-compatible revisions. Preserve failed-job/call evidence, revoke affected credentials where needed, and keep database downgrade/reset outside automatic rollback. A provider login problem should not trigger disabling authorization or exposing Supabase tables.

Sources: [Render free services](https://render.com/docs/free), [Blueprint reference](https://render.com/docs/blueprint-spec), [Docker startup](https://render.com/docs/docker), [Supabase connections](https://supabase.com/docs/guides/database/connecting-to-postgres), [database roles](https://supabase.com/docs/guides/database/postgres/roles), [Data API security](https://supabase.com/docs/guides/api/securing-your-api), [Google provider](https://supabase.com/docs/guides/auth/social-sign-in/auth-google), [redirect matching](https://supabase.com/docs/guides/auth/redirect-urls), [Supabase Free pricing/quotas](https://supabase.com/pricing), [Vercel project settings](https://vercel.com/docs/project-configuration/project-settings), [Vercel environment scopes](https://vercel.com/docs/environment-variables), [Vercel Hobby terms/limits](https://vercel.com/docs/plans/hobby).
