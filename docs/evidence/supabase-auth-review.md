# Supabase Google authentication implementation evidence

Date: 2026-10-25 (Africa/Johannesburg). Input revision: `4332231bcfdcef57ece61283a1b1ad171fa3df28`.

This change is limited to the identity package, the server login provider dispatch, the Supabase browser-session test, and these evidence files. It preserves local mock and legacy Google authentication. It changes no database schema, frontend, infrastructure, dependency, or cloud resource.

## Protocol and trust boundaries

- `AUTH_MODE=supabase` requires `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY`, and server-only `SUPABASE_SECRET_KEY`. Modern `sb_secret_` keys use the `apikey` header; legacy `service_role` JWT keys additionally use `Authorization: Bearer`. Keys are never placed in browser URLs.
- The configured callback shares `APP_ORIGIN`, normally `API_PUBLIC_URL=<APP_ORIGIN>/api`, yielding `/api/v1/auth/callback` through the Next.js proxy. Go still handles `/v1/auth/callback` internally. Production requires HTTPS and rejects loopback Supabase endpoints.
- Supabase `/auth/v1/authorize` receives `provider=google`, `scopes=openid email profile`, and PKCE `code_challenge_method=s256`. OptiTrace's random state is inside the `redirect_to` callback query; it does not override Supabase's upstream OAuth state.
- The existing database transaction stores hashed local state and the PKCE verifier for five minutes. The browser carries an HttpOnly SameSite=Lax state cookie. Callback state must match that cookie and consume the database transaction atomically. The existing nonce column remains available for direct OIDC; Supabase owns its upstream OIDC nonce, so the local nonce is not claimed to be a Supabase-validated ID-token nonce.
- Code exchange uses `POST /auth/v1/token?grant_type=pkce` with JSON `auth_code` and `code_verifier`. The code/verifier pair is single-use. The whole exchange has a twenty-second service deadline.
- Identity comes from authenticated Supabase `/auth/v1/user`, never from unsigned JWT payloads or the token response's embedded user. Require a stable UUID, `authenticated` audience, confirmed email, a verified Google identity, and no ban/deletion/anonymous flag.
- Also require the Google provider access token from this PKCE exchange to authenticate at the fixed Google UserInfo endpoint. Its verified email and `sub` must match the Supabase user's `google` identity's explicit `identity_data.sub`. This rejects email-only sign-ins and accounts that merely have a historic Google link without a current Google OAuth credential.
- Ban/deletion status is checked through admin `GET /auth/v1/admin/users/{uuid}` at login and at the existing maximum five-minute session-check interval. Admin lookups do not depend on a user access token remaining valid for the application's eight-hour session.
- Supabase access/refresh and Google provider tokens are transient and discarded. The database `provider_token` field is empty for Supabase sessions. Only the hash of the opaque OptiTrace session secret is persisted. Existing invitation, membership, CSRF, origin, revocation, and eight-hour expiry rules remain in Go/PostgreSQL.
- All Supabase and Google verification clients refuse redirects. Provider bodies and token-bearing URLs are not included in authentication errors or logs.

## Verification

Native Go tests run against an isolated cached PostgreSQL 18 test container; the running application database was not touched. Native `CGO_ENABLED=0` cannot perform race detection, so the existing cached Linux race-builder image was used without building an image.

- `go test ./... -count=1`: exit 0, including both legacy OIDC/mock PostgreSQL integration and the Supabase browser-session integration test.
- `go vet ./...`: exit 0.
- `git diff --check`: exit 0.
- Linux `go test -race ./internal/identity ./internal/server -count=1 -v`: exit 0, including both database-backed authentication flows.
- Logs: `supabase-auth-tests.log` and `supabase-auth-race.log` beside this file.

The new tests cover PKCE exchange, wrong verifier, single-use provider code, local state/cookie mismatch, local state replay and expiry before token exchange, callback across API instances, uninvited accounts, stable UUID membership binding, ignored untrusted token-response user fields, no retained provider credentials, ban/deletion and admin outage/missing-user rejection, explicit Google subject/email verification, redirect credential containment, hosted configuration rejection, CSRF/origin checks, and eight-hour opaque session issuance.

## Remaining hosted acceptance

No managed Supabase project, Google OAuth client, Vercel deployment, or Render deployment was configured or exercised. Protocol doubles are synthetic verification, not proof of hosted authentication.

Google's OAuth callback must be `<SUPABASE_URL>/auth/v1/callback`. Supabase's redirect allowlist must allow the fixed frontend callback path plus its random `state` query. Use a fixed production host/path and the documented escaped-query glob, for example `https://fixed-host/api/v1/auth/callback\?state=*`; actual managed-provider acceptance remains a hosted test. Avoid wildcard hosts and preview deployment authentication.

App logout revokes the local OptiTrace session. Provider account bans/deletions are bounded by the five-minute check interval; signing out of a separate provider session alone is not represented as revocation of all independent OptiTrace sessions. Existing Identity Platform user IDs must not be silently relinked by email to new Supabase UUIDs; an existing deployment needs an explicit account migration, or a fresh application database and invitations.

## Primary references inspected

- [Supabase PKCE flow](https://supabase.com/docs/guides/auth/sessions/pkce-flow).
- [Supabase Auth REST OpenAPI](https://github.com/supabase/auth/blob/master/openapi.yaml): authorize, PKCE token, authenticated user, and admin user endpoints.
- [Supabase PKCE server implementation](https://github.com/supabase/auth/blob/master/internal/api/token.go): provider credentials are returned from the completed flow state, and the flow state is consumed.
- [Supabase API keys](https://supabase.com/docs/guides/getting-started/api-keys): modern secret keys belong on `apikey`, not bearer authorization, and remain server-only.
- [Supabase Google sign-in](https://supabase.com/docs/guides/auth/social-login/auth-google).
- [Google OpenID Connect and UserInfo](https://developers.google.com/identity/openid-connect/openid-connect).
