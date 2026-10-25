# Portfolio hosting integration evidence

Recorded 2026-10-25. Input `4332231`; authentication integrated as `897e594` and hosting as `30ea0d4`. Runtime/database/proxy integration is the companion implementation commit. No hosted resources or remote repository were created by this work.

Final application implementation revision: `eb00a42`; documentation clarification: `a7a1b24`. The final serialized race run passed all four affected packages, exit0. Evidence-only additions do not change the tested binaries.

## Changes and review

Added `cmd/portfolio` with one public API listener, loopback-only synthetic adapters and a durable polling worker. Hosted startup rejects mock identity, migration credentials, external adapters/cloud jobs/buckets, non-application database logins and unverified TLS. Pool maximum is six. Shutdown forcibly closes active HTTP requests after its grace period before closing the database pool.

Migration003 adds RLS-protected append-only per-call rows and explicit Supabase Data API role denials on application objects. The old row-level JSON journal update is removed; list/detail responses merge legacy and new journal entries for unconfirmed attempts. All six capability/phase events commit while the original attempt row holds a `FOR NO KEY UPDATE` lock. This removes that shared-row bottleneck, not all possible persistence or latency failures.

The auth engineer independently reviewed integration source. Their two findings were corrected: graceful shutdown alone left active request contexts alive; SQL CHECK allowed missing/null capability because SQL NULL passes CHECK. Regression tests cover both fixes. Independent final acceptance remains distinct from this focused review.

Next's proxy has a 55-second upstream timeout and 60-second route duration, returns structured 503/Retry-After for successful HTML wake pages, preserves login redirects/cookies and never retries a mutation automatically.

## Observed local checks

| Command/check | Result |
|---|---|
| Native `go test ./... -count=1` with disposable PostgreSQL and both auth test URLs | Exit0, all packages; mock and Supabase protocol/browser sessions, migrations and RLS exercised |
| Native `go vet ./...` | Exit0 |
| After final config/benchmark guard changes: `go test ./internal/server ./internal/storage ./internal/portfolio -count=1` | Exit0; raw `.artifacts/portfolio-native.log` |
| `go build -o ../../.artifacts/portfolio.exe ./cmd/portfolio` | Exit0 |
| `npm test` | Exit0, 15/15 tests |
| `npm run typecheck`, `npm run format:check` | Exit0 |
| `npm run build` | Exit0, Next16.4.2 production build |
| `Test-PortfolioConfig.Tests.ps1` / static validator | Exit0; valid synthetic configuration plus14 unsafe configurations; pool cap aligned with Go |
| `docker build -f infra/portfolio.Dockerfile -t optitrace-portfolio:local .` | Exit0; raw `.artifacts/portfolio-image-build.log` |
| Native combined runtime | Health200, public `/authorize`404; SQL job completed once with persisted result |
| Built container runtime | Same health/private endpoint checks; queued benchmark completed once with persisted result; SIGTERM exit0, OOM false |
| Hosted mock image startup negative check | Expected exit1 before database connection |

Image inspected locally: non-root `10001:10001`, exec CMD `/app/portfolio`, no ENTRYPOINT, size39,806,091 bytes. Local image manifest `sha256:343de4b4c4d3c6dac0fa9e83d24c4fc72eeb5245e3fa83fc89f605e337d835ce`; this is not a published registry digest or deployment record.

## Race checks and limitations

First Linux run used host networking, which did not reach the Windows-published test database port; database-dependent packages failed connection-refused. Corrected the test network to Docker bridge and the isolated database's container address. The next combined run passed identity/storage/portfolio but failed the existing server workflow under simultaneous compilation/build load. Its isolated rerun passed, exit0, in `.artifacts/portfolio-race-server.log`. No race report was observed; this does not establish a cause for the first server workflow failure. A final serialized run is recorded separately below.

Final command: cached Linux Go1.26.10/GCC race builder, `go test -race -p 1 ./internal/portfolio ./internal/storage ./internal/identity ./internal/server -count=1`, Docker bridge network, both test database URLs pointing at the isolated PostgreSQL container. Exit0: portfolio1.220s, storage1.607s, identity1.316s, server2.772s, with no race reports. See `portfolio-integration-race.log`; cold compilation time is not a performance measurement.

Only disposable synthetic PostgreSQL was mutated. The primary local Compose stack was not migrated/rebuilt by this task and still runs its previous image. Temporary runtime smoke containers were removed; the test database container can be stopped after final checks, preserving its data. Context/ and the user's untracked logo artifacts remain untouched.

Real Supabase/Google redirect acceptance, managed database ownership/grants/TLS, free quotas, hosted deployment and full browser/performance acceptance remain open. The earlier HOLD in `docs/evidence/NEXT-PHASE.md` is not silently cleared. No new warm API performance result or MTN saving is claimed. Hosting/account access and release scope are recorded in the portfolio release proposal; tokens and verification codes are excluded from evidence.
