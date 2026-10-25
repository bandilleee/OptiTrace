# ADR0004: Free portfolio hosting with Go-managed Google sessions

Status: accepted by user, 2026-10-25. Deployment remains subject to concrete release approval and provider acceptance.

## Decision and context

GCP billing activation is currently unaffordable. Use existing Vercel, Render and Supabase accounts for a synthetic personal portfolio deployment. Vercel hosts Next.js and its same-origin proxy. One Render Free Docker service runs Go API, a loopback-only simulator and the durable benchmark lease worker. Supabase Free supplies PostgreSQL and Google authentication. Retain optional GCP Terraform and Identity Platform support; do not reuse unrelated applications or their resources.

Supabase manages upstream Google OAuth. Go generates its own expiring state/PKCE transaction, consumes it once, exchanges the code and verifies identity with authenticated Supabase `/user` plus Google UserInfo. The verified Google subject must match the linked identity. Invitations bind the stable Supabase UUID. Go stores only hashed opaque application sessions, never upstream credentials. Admin lookups enforce provider account changes with a maximum five-minute cache. An upstream identity alone grants no tenant access.

The API uses a dedicated non-owner `optitrace_login` with no elevated inherited permissions, verified TLS and IPv4 session pooling. Migration credentials are separate. Disable Data API exposure and revoke its default roles' access to application objects, including auth helper functions. Append-only per-call event rows replace competing writes to one attempt JSON row; intentions commit before calls, completions afterward. Failed or ambiguous execution retains its idempotency reservation.

## Consequences and alternatives

The single process avoids a paid background worker. Jobs persist before launch, use leases/fencing and resume on a later wake; an idle sleeping service cannot promise immediate completion. Render sleep, possible Supabase inactivity pauses, free quotas and cold-start delays are explicit demo limits. They do not satisfy production uptime or establish the warm 500ms performance gate. Portfolio benchmarks cap at 100 cases. No self-ping or paid resources are introduced.

Vercel proxy requests have a 55-second upstream budget within a 60-second route duration and return structured unavailable responses for host wake pages. Mutations are never automatically replayed. Existing local mock authentication remains development-only; hosted configuration rejects it. No Firebase SDK or emulator is introduced.

GCP would provide the original regional topology but requires active billing. Hosting only Next.js would omit the Go runtime and workers; exposing Supabase tables directly would bypass the intended authorization boundary. These alternatives are not selected.

## Acceptance

Local protocol/auth, PostgreSQL/RLS, migration, role bootstrap, proxy, worker and shutdown checks precede a release. Actual Google redirect allowlists, hosted admin-account checks, managed database grants, pooler TLS, worker recovery, authenticated end-to-end smoke tests and performance remain hosted acceptance requirements. See `docs/runbooks/PORTFOLIO-HOSTING.md` and migration evidence; local tests are not hosted success.
