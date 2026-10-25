# OptiTrace portfolio preview release proposal

Prepared 2026-10-25. This document is reviewable scope, not a deployment record.

## Observed accounts and proposed resources

All three provider CLIs authenticated successfully. Read-only account checks found:

| Provider | Selected account scope | Proposed new resource |
|---|---|---|
| GitHub | `bandilleee`; no existing `OptiTrace` repository found | Private `bandilleee/OptiTrace`, push reviewed local main |
| Supabase | Existing `OptiTrace` organization `znwkartuhyniucsgtstm`; no project listed in it | Free project `optitrace`, Frankfurt (`eu-central-1`), separate owner/application credentials |
| Render | `My Workspace` (`tea-d6gnjohdrdic73caidl0`) | New Free `optitrace-api` service in Frankfurt, one portfolio process, manual deployments |
| Vercel | `echoverses-projects`, observed Hobby plan | New `optitrace` project rooted at `apps/web`, Node22, manual first deployment |

Use the provider-assigned fixed Vercel production URL after project creation for exact Go/Supabase origin and callback configuration; availability of a short alias is not assumed. Render's assigned HTTPS URL becomes server-only Vercel `API_BASE_URL`. No application credentials enter frontend public variables.

## Authorized release scope requested

Create only these dedicated resources on free plans; stop if a provider requires payment, an upgrade or unavailable quota. Do not change, delete, resume or reuse existing projects/services. Do not add a worker service, database service on Render, disk, cron, paid domain or self-ping. Keep the repository private and automatic Render deployment off.

Bootstrap the isolated Supabase application role, apply reviewed migrations without development seeding, disable Data API exposure, configure Google authentication, provision the first invited administrator through the existing operator workflow, and deploy the reviewed image/web bundle. Account/database secrets remain in local credential storage and provider secret settings; do not print or commit them. Google web OAuth credentials and consent/test-user setup are prerequisites; they are separate from paid GCP workload billing. Confirm the user's intended administrator Google account before provisioning membership.

Inspect the deployed application through authenticated smoke tests: invitation binding, denied uninvited user, CSRF, tenant isolation, policy approval, evaluation/trace, benchmark persistence and recovery, exports, logout and provider-disable checks. Verify actual managed database privileges, pooler TLS, redirect allowlists, image digest and rollback. Public availability of a login page alone is not acceptance.

## Known limitations and stop conditions

This is a synthetic portfolio preview, without a production uptime claim. Render can sleep; Supabase can pause inactive projects; worker progress requires an awake process. Provider free quotas and managed PostgreSQL version must be checked at creation. No spending beyond free allocations is authorized.

Existing performance and live-browser acceptance failures in `docs/evidence/NEXT-PHASE.md` remain open until rerun successfully. The new append-only journal removes a demonstrated row-lock contention mechanism; it does not prove the 500ms/relative-p95 gates. Real Google/Supabase sign-in and hosted end-to-end acceptance are not yet observed. Do not label simulated charges as MTN savings.

Rollback means rolling Render back to the preceding verified image/revision and redeploying the preceding Vercel bundle. First release has no known-good hosted revision; stop public acceptance on a failed smoke test. Take a private backup before migration and rehearse restoration. Do not reverse migration003 by dropping evidence rows or weakening grants; use a reviewed forward fix.

## Required approval

Root `AGENTS.md` requires: "Do not deploy or mutate cloud resources without release approval." Approval of this proposal permits dedicated free preview provisioning/deployment and its acceptance checks; it does not authorize paid resources, unrelated project changes or production acceptance claims.
