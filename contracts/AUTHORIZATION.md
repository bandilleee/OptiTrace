# Frozen authorization and API boundary

Input: `b39abee7dbd3258fc793050be773305827dbbac0`. This is an implementation contract, not a claim of independent security acceptance.

## Principal and tenant

Go authenticates the opaque `optitrace_session` cookie or an `Authorization: Bearer` integration credential. Browser identity comes from configured Google Cloud Identity Platform REST after OIDC state, nonce, PKCE and provider validation. No Firebase SDK/emulator is permitted. The existing protocol test provider is restricted to localhost development and is not evidence of hosted Google authentication. Production must reject mock identity, localhost providers and incomplete configuration.

An upstream account alone grants no tenant access. Invitation acceptance binds verified identity to membership. Go looks up current tenant roles for each session request. The `{tenant}` path is a requested resource scope, not proof of membership; body/header tenant assertions cannot grant access. Keys resolve a stored tenant and scopes, must match the requested tenant, and cannot impersonate a browser member. Invalid/revoked credentials fail closed; do not fall back to a cookie after an invalid supplied bearer credential.

## Session permission matrix

Roles combine by union; unlisted actions are denied. `tenant_admin` is not a superuser.

| Resource/action | Allowed tenant roles |
|---|---|
| Read policies | policy_author, policy_reviewer, integration_developer, auditor |
| Create policy / submit own draft | policy_author |
| Approve / activate | policy_reviewer, with author != actor |
| Read evaluations / benchmarks / export | integration_developer, auditor |
| Execute evaluation / enqueue benchmark | integration_developer |
| Read/change memberships, read/create invitations, read/create/revoke credentials | tenant_admin |
| Read audit | auditor |

`platform_operator` is reserved for separately authorized provisioning CLI operations, not assignable through tenant invitations/membership changes and grants no implicit browser access. Tenant roles accepted by those mutations are the other five values only. Do not remove the final tenant administrator. A credential may carry only `evaluations:read` and/or `evaluations:write`: read allows evaluation list/detail; write allows evaluation creation. Neither scope grants policies, benchmarks, auth/me, logout, audit or administration. Keys inherit no user roles. Plaintext secrets appear only at creation and must never enter logs/audit/export; persist hashes and enforce revocation.

## Transport and persistence

The browser uses same-origin `/api/v1/*`; Next proxies to Go `/v1/*`. Go owns authorization even when requests bypass Next. Cookie mutations require both exact `Origin=APP_ORIGIN` and matching `X-CSRF-Token`. Automated integration credentials are exempt on their permitted evaluation mutations only. Sessions expire after eight hours and provider-disabled checks have an existing maximum five-minute recheck interval; membership checks are current per request. Secure cookies may be disabled only for localhost development.

PostgreSQL applies transaction-local tenant context only after authorization. Tenant reads/writes must obey RLS (`USING` and `WITH CHECK`); pooled connections cannot retain context. The runtime login must not own tables or possess superuser/BYPASSRLS/inherited elevated roles. Migrations use a separate identity. Narrow security-definer auth and worker functions have fixed search paths, no PUBLIC execute, and bounded purpose. Database tenant isolation does not replace Go role checks; a database login is not an end-user role. Authentication/provisioning tables and cross-tenant helper functions need separate privilege review.

Writes and audit evidence commit atomically before success responses. Durable evaluation attempts/journals are deliberately separate transactions so a failed request does not erase possibly billable work. Workers claim durable jobs with leases and fencing tokens; stale workers must not overwrite completed results. Queue retries are distinct from evaluation/source-call replay.

## API behavior and errors

Use [openapi.json](openapi.json) for paths and JSON shapes; snake_case and `{error:{code,message}}` remain fixed. All tenant paths require authenticated authorization, including list/detail/export. `401` means invalid/absent authentication; `403` means denied tenant/role/scope or CSRF. A foreign object within an authorized tenant resolves as `404`. Invalid input is `400`; illegal transitions, self-approval, last-admin removal and idempotency conflicts are `409`; bounded queue pressure is `429`. Infrastructure failure is a non-success response, never a fabricated reassuring evaluation. Exact error codes already emitted remain compatible.

Policy creation is `201`, transitions `200`; evaluation first completion `201`, persisted replay `200`; benchmark enqueue `202`; invitation/credential creation `201`; membership update and credential revocation `200`; logout `204`. Lists use `{items:[]}`. Policy bodies cannot set tenant, author, reviewer, status or version. Policy GET by ID is supported; no edit/delete/retire endpoint is exposed.

Evaluation creation requires `Idempotency-Key` (1..200 bytes; ASCII keys recommended), scoped by tenant and bound to the decoded request hash. An omitted strategy defaults to `optitrace`; current hashing occurs before this default, so omitted and explicit defaults can conflict with the same key. Concurrent duplicate calls must never dispatch another billable execution. An existing unfinalized reservation returns `409 execution_unconfirmed`; discover retained attempts through the evaluation list/detail endpoints. There is no automatic replay TTL in this contract.

Benchmark omitted/zero count defaults to 50; explicit nonzero count must be 2..1000. Omitted/zero seed defaults to 42; nonzero signed int64 seeds are accepted. At most three queued/running jobs per tenant. Export returns the benchmark envelope including report/results, not a bare report. Preserve queued/running/completed/failed and error text. Synthetic supplier prices, virtual timing, measured engine HTTP timing and measured authenticated API timing are separate evidence categories. No assumed price or simulated saving is an observed MTN result.

## Required integration evidence

Integrator review must cover contract changes and regenerated client types in the frontend-owned path. Acceptance requires disposable PostgreSQL tests under a non-owner runtime role (cross-tenant list/detail/write, immutable policies, role denial, concurrent idempotency, last-admin protection), real authenticated API journeys and Linux race checks. Documentation/schema checks alone establish none of these runtime guarantees. Hosted Identity Platform and cloud release remain separately approved work.
