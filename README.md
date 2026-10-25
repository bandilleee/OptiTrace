# OptiTrace

An invitation-only verification operations console and a bounded network-API execution optimiser. Next.js/TypeScript, Go, embedded OPA and PostgreSQL. The selected portfolio deployment uses Vercel, Render Free and Supabase PostgreSQL/Auth with Google sign-in; Go owns application sessions and authorization.

The included South Africa workloads, capabilities, prices and policy thresholds are **synthetic assumptions**. The application does not establish MTN savings or real fraud-detection performance. A benchmark that finds no improvement remains visible.

## Portfolio deployment

For hosted setup, follow [the portfolio hosting runbook](docs/runbooks/PORTFOLIO-HOSTING.md). One Render process runs the API, private loopback simulator and durable benchmark worker. Supabase provides Google authentication and PostgreSQL, with a dedicated non-owner application login and no Data API access to application tables. Vercel proxies browser requests to Go; provider tokens never become browser application sessions.

The invitation-only [portfolio preview is live](https://optitrace.vercel.app). [Hosted release evidence](docs/evidence/PORTFOLIO-PREVIEW.md) records real Google login, distinct author/reviewer policy approval, evaluations, persisted traces and the benchmark popup. Full production acceptance remains HOLD.

Render Free can sleep and Supabase Free can pause inactive projects. Cold starts and paused services are availability limits, separate from warm API performance gates. Portfolio benchmarks are capped at 100 cases. The optional GCP infrastructure remains available for a later separately approved release.

## Run locally on Windows

Requirements: PowerShell7, Docker Desktop with Linux containers, Node22+, and Go with automatic toolchain selection enabled. The project pins Go **1.26.10** because the originally installed1.25.5 has known reachable vulnerabilities. Go downloads the project toolchain automatically without changing your global installation.

```powershell
Set-Location C:\Projects\OptiTrace
./scripts/Doctor.ps1

# Containers for the entire application:
./scripts/Dev.ps1 up

# Alternatively, native Go/Node with PostgreSQL in Docker:
./scripts/Start-Native.ps1
```

Open [the console](http://localhost:3000). Select **Continue with Google**, then an invited identity on the clearly labelled local protocol test provider. This exercises OIDC discovery, signing, authorization code, state, nonce, PKCE and Identity Platform-style REST; it is not a Firebase emulator or a password bypass.

| Synthetic identity | Roles |
|---|---|
| `admin@optitrace.test` | Tenant administrator, integration developer, auditor |
| `author@optitrace.test` | Policy author, integration developer |
| `reviewer@optitrace.test` | Policy reviewer, auditor |
| `auditor@optitrace.test` | Read only |
| `outsider@optitrace.test` | A separate isolated tenant |
| `uninvited@optitrace.test` | Sign-in denied |

Use either container or native application processes, not both on the same ports. `./scripts/Start-Native.ps1 -Action stop` stops only verified processes it owns; PostgreSQL data is retained. For containers, `./scripts/Dev.ps1 down` preserves the database volume.

Stop native services before production-build checks on Windows: Node holds its standalone output directory open. Container mode leaves the host build directory unlocked.

## Try the workflows

1. As admin, run both policy profiles with clear, adverse and unavailable observations. Inspect the persisted checks, timestamps, compulsory work, configured charges and stopping reason.
2. As author, create and submit a draft. Sign in separately as reviewer to approve and activate it. Self-approval is prohibited.
3. Launch a seeded benchmark, wait for the durable worker and compare all five strategies. Virtual estimates, measured engine HTTP samples and complete-API measurements are separate evidence categories. Inspect/export the complete JSON report for gates and uncertainty.
4. Create a scoped integration credential, copy its one-time secret, and revoke it. Invite users and inspect audit history. The auditor cannot perform mutations.

Started source calls are journaled durably. If execution is interrupted, its trace and key remain unconfirmed; the same idempotency key is never automatically repeated. A journal starting event is a possible call, not confirmation of a charge. Final results and approved policy versions are immutable.

## Verification

```powershell
./scripts/Check.ps1 -Scope All
./scripts/Test-Harness.ps1
./scripts/Generate-Client.ps1 -Check
./scripts/Generate-Storage.ps1 -Check

# Start a DISPOSABLE database; tests migrate and seed it. Port 55432 must be free.
docker run --name optitrace-tests --rm -d -p 127.0.0.1:55432:5432 -e POSTGRES_PASSWORD=local-test postgres:18
$env:TEST_DATABASE_URL='postgres://postgres:local-test@localhost:55432/postgres?sslmode=disable'
./scripts/Check.ps1 -Scope Engine -RequireDatabase
docker stop optitrace-tests

# With the application running:
$env:OPTITRACE_LIVE='1'
$env:PLAYWRIGHT_BASE_URL='http://localhost:3000'
npm --prefix apps/web run test:e2e
./scripts/Measure-Api.ps1 -Samples 100
```

Go tests cover bounded policy equivalence, 13 execution plans, adversarial observations, cancellation/journaling, protocol authentication and PostgreSQL authorization. Web tests cover client/proxy boundaries and real browser journeys. GitHub Actions also configures Linux race checks, vulnerability/image scans, code-generation drift, Windows checks and Terraform validation/mock plans. The private repository is connected, but GitHub account billing restrictions currently prevent Actions jobs from starting; hosted CI remains unverified.

`npm --prefix apps/web test` runs unit tests. Without `OPTITRACE_LIVE=1`, browser tests use controlled API fixtures to exercise UI states. With it, the separate live suite uses the actual local Go API, database and identity provider.

## Authentication and optional GCP profile

Selected hosted identity uses Supabase Auth with Google PKCE. Verified invitations bind stable Supabase UUIDs; PostgreSQL memberships determine authorization. Go issues opaque eight-hour sessions, stores their hashes, enforces CSRF/Origin checks and rechecks disabled provider accounts at most every five minutes. API keys have separate scopes and revocation. Google Cloud Identity Platform REST remains available for the optional GCP profile.

Copy `.env.example` to an ignored `.env` for manual native development and use `scripts/Import-LocalEnv.ps1`. Portfolio hosted secrets belong in Render/Supabase settings; Vercel receives only its server-side API base URL. Real Google sign-in needs an approved OAuth client in Supabase and exact registered callbacks. Secret Manager and Identity Platform keys apply only to the optional GCP profile. No cloud credentials are needed for the default synthetic environment.

Terraform maps the API, console and private simulator to Cloud Run, benchmarks to Cloud Run Jobs, persistence to Cloud SQL, reports to private Cloud Storage, and delivery identity to Workload Identity Federation. Defaults use Johannesburg regional compute/database resources, with production HA and deletion protection. This optional GCP workload deployment has not been performed; the selected portfolio preview runs on Vercel, Render and Supabase.

Read `docs/ARCHITECTURE.md`, `docs/runbooks/RELEASE.md`, `docs/runbooks/AGENT-HARNESS.md`, `policies/README.md` and `benchmarks/README.md` for contracts and evidence boundaries. The release runbook requires explicit approval, current cost inputs, real-provider smoke tests, backup restoration and rollback rehearsal before hosting acceptance.

Observed local results and remaining release prerequisites are recorded in [integration verification](docs/evidence/verification.md).

The historical [parallel delivery and acceptance handback](docs/evidence/NEXT-PHASE.md) records evaluator/planner repairs and the full acceptance **HOLD**. The [hosted preview report](docs/evidence/PORTFOLIO-PREVIEW.md) updates Google login and hosted functional checks; full controlled performance, independent acceptance and recovery drills remain open. Historical passing checks do not override unresolved failures.

The [portfolio integration evidence](docs/evidence/portfolio-integration.md) records the subsequent hosting/authentication migration and local checks. Its passing component checks do not clear the earlier full acceptance HOLD.

The approved frontend uses shadcn/ui, Tailwind4, local Montserrat fonts, and the supplied animated logo kit. See the [design implementation notes](docs/design/FRONTEND.md) for the palette, component mappings, prototype adaptations, and browser checks.
