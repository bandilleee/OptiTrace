# Google Identity Platform acceptance

Status on 2026-10-23: **not executed**. There is no approved OptiTrace Google project or configured OAuth client. The signed-in gcloud default belongs to a different application and must not be reused. The only visible billing account is closed. These instructions do not authorize project, provider, IAM or account mutations.

## Configuration packet

Use one isolated development project with the Google provider enabled in Identity Platform. Supply the following through a private process environment or Secret Manager, never Git, screenshots, command history, browser storage or handback logs:

| Setting | Local Google profile | Hosted staging |
|---|---|---|
| `AUTH_MODE` | `google` | `google` |
| `APP_ENV` | `development` | `production` |
| `APP_ORIGIN` | `http://localhost:3000` | Approved exact HTTPS DNS origin |
| `API_PUBLIC_URL` | `http://localhost:3000/api` | Approved HTTPS origin plus `/api` |
| `AUTH_COOKIE_SECURE` | `false` for localhost HTTP only | `true` |
| `GOOGLE_CLOUD_PROJECT` | Approved isolated dev project ID | Approved staging project ID |
| `GOOGLE_CLIENT_ID` | Web OAuth client in that project | Staging web OAuth client |
| `GOOGLE_CLIENT_SECRET` | Private value | Secret Manager version |
| `IDENTITY_PLATFORM_API_KEY` | Key restricted to the Identity Toolkit API | Secret Manager version |
| `DATABASE_URL` | Dedicated local non-owner application login | Non-bypass Cloud SQL application login |
| `SIMULATOR_URL` | Local synthetic simulator | IAM-protected simulator URI |

Register exactly `http://localhost:3000/api/v1/auth/callback` locally and `<APP_ORIGIN>/api/v1/auth/callback` when hosted. Declare exact authorized domains in Identity Platform, and add approved test accounts to a Google OAuth consent configuration in testing mode. No Identity Platform tenants are required. Real Google authentication still uses synthetic verification evidence.

For local testing, stop the mock API/web and run native Go/Node with the above environment; `scripts/Start-Native.ps1` always forces mock mode and is **not** the real-Google launcher. Build/run `./cmd/api`, `./cmd/simulator` and `./cmd/worker` from `services/engine`; run the Next.js web with `API_BASE_URL=http://localhost:8080`. Give the simulator `AUTH_MODE=google` so it does not expose mock identity endpoints. Do not run `-seed` for hosted bootstrap. Keep the real-identity database isolated from demonstration memberships.

Provider account-status checks use Application Default Credentials and `firebaseauth.users.get` on this project. This is a Google IAM permission name, not a Firebase SDK dependency. Local gcloud CLI login alone does not guarantee ADC is configured. Arrange approved user ADC or service-account impersonation; do not create/download a service-account key. Cloud Run uses its attached service account. Run migrations with the distinct migration principal, then use `cmd/admin` with an operator database connection to create the first tenant invitation. The first Google login binds the invitation to the stable Identity Platform `localId`.

## Expected network path

1. Browser requests `/api/v1/auth/login` through Next.js.
2. Go saves state, nonce and PKCE verifier and redirects to Google's authorization endpoint.
3. Google returns an authorization code to the exact callback. Go exchanges it, verifies issuer/audience/expiry/nonce and verified email, then POSTs the Google ID token as URL-encoded `postBody` with `providerId=google.com` to `accounts:signInWithIdp`.
4. Go accepts the verified account only with an invitation or existing membership and creates its own opaque session. Browser receives no provider refresh token. An Identity Platform account without membership still has no application access.
5. Browser uses the HttpOnly session and CSRF token; API keys are a separate integration path. No Firebase SDK or emulator participates.

The request/response shapes are defined by [Identity Platform REST](https://docs.cloud.google.com/identity-platform/docs/reference/rest/v1/accounts/signInWithIdp) and [the REST guide](https://docs.cloud.google.com/identity-platform/docs/use-rest-api). The app implementation is `services/engine/internal/identity/identity.go` and `internal/server/server.go`.

## Acceptance on an approved stable staging origin

Use two disposable Google accounts and two tenants with synthetic evidence. Record candidate commit, immutable API/web/simulator/worker digests, region, callback origin, migration execution and reviewer. Never save authorization codes, cookies, session hashes, ID/access tokens, CSRF values, API key plaintext or OAuth secrets in evidence.

| Case | Action and required observation |
|---|---|
| Invited identity | Sign in with invited verified Google account; see expected workspace/roles. Confirm membership binds `localId`, not just email. |
| Uninvited identity | Sign in with a different real Google account; no usable application session or tenant evidence. Account auto-creation in Identity Platform alone grants no application permission. |
| Redirect/transaction | Reject a callback with altered state, missing transaction cookie, and replayed callback; do not disclose code/state in reports. |
| OIDC/PKCE negative cases | Keep protocol-level signed wrong-issuer/audience/nonce/expiry and invalid-PKCE tests passing; Google cannot legitimately issue those malformed fixtures. Do not claim the fixtures are real-provider execution. |
| Session/cookie | Confirm HttpOnly, Secure, SameSite=Lax and eight-hour expiry; JavaScript cannot read the cookie. Verify logout causes the next authenticated API call to fail. Test expiry using a disposable test session and authorized local test clock/DB fixture; never weaken hosted expiry. |
| CSRF | Same-origin mutation succeeds with valid CSRF; missing/invalid token and wrong Origin fail. |
| Provider disable | Disable the disposable Identity Platform user through an approved operator action. Existing session must be rejected on first check after the five-minute maximum cache. Re-enable/clean up only with approval. |
| Membership removal | Remove the test user's membership through an authorized administrator; the next request loses tenant access without waiting for provider cache expiry. |
| Tenant isolation | Tenant A's browser session/API credential cannot read or modify Tenant B resources, including traces/reports/export URLs. |
| Workflow | Separate author/reviewer create, submit, approve and activate; own-version approval fails. Developer evaluates and sees a persisted trace; auditor can read but cannot execute. |
| Jobs/report | Submit bounded benchmark, inspect queued/running/completed popup and authenticated export; simulate a disposable worker interruption and observe lease recovery without duplicate completion. |
| Keys | One-time scoped key can evaluate within tenant, cannot administer membership; revocation takes effect. Browser session is not required for authorized integration keys. |
| Hosted test endpoints | Mock `/authorize`, `/token`, `/jwks` and `/identity/` routes are absent; production refuses `AUTH_MODE=mock`. |

Use DevTools only to inspect status/header attributes locally; redact sensitive values before any screenshot. Normal users complete the journeys in the application.

## Tagged revision limitation

A tagged web revision still proxies to its configured `API_BASE_URL`, normally the stable API URI. Its OAuth callback also targets the registered stable origin. Therefore a green tagged web page plus a green tagged API `/healthz` is **not** proof of a paired candidate login or authenticated journey. Accept the pair on an isolated staging environment's registered stable origin after staging-only traffic approval, or deliberately configure and register a complete candidate origin/API pair. Never change the production OAuth callback merely to make a tagged smoke check pass.

Capture a redacted evidence packet with explicit pass/fail/not-run for each row. Real-provider acceptance remains separate from local mock tests and from final production promotion approval.
