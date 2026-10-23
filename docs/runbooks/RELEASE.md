# Release and rollback

Cloud writes require explicit release approval. No project, OAuth client, registry or database is created by running local development/tests.

## Prerequisites and identity

Follow [the isolated bootstrap proposal](CLOUD-BOOTSTRAP.md) before creating an OptiTrace project. Select an explicitly approved project and open billing account; never inherit another application's gcloud default. Create a dedicated development project for optional real Google sign-in tests. Configure Google OAuth consent and exact redirects: `http://localhost:3000/api/v1/auth/callback` locally and `<approved HTTPS origin>/api/v1/auth/callback` in hosting. Enable the Google provider in Identity Platform. Keep client secrets in the approved secrets workflow. Invite the initial operator/admin via the privileged administrative CLI; never enable the synthetic seed in hosting.

Application tenants are PostgreSQL entities; v1 does not use Identity Platform tenants. Google and Identity Platform account/linking errors fail sign-in rather than silently linking accounts. Complete [real Google acceptance](GOOGLE-ACCEPTANCE.md) on a dedicated project before accepting a cloud release. Local protocol tests are necessary but are not live-provider acceptance.

## Infrastructure review

1. Run all local/CI checks and independent security acceptance against the candidate commit. Review dependency and image findings.
2. Build/publish immutable image digests using a protected release workflow and Workload Identity Federation. Do not use service-account JSON keys. The Terraform trust condition binds immutable numeric repository/owner IDs, the exact repository name, main branch and protected GitHub environment. Obtain both numeric IDs from the approved repository. Configure GitHub environment reviewers and deployment branch protection; Terraform does not configure those GitHub controls. Set the staging-only protected variables `GCP_OPTITRACE_STAGING_PROJECT_ID` (the isolated approved `optitrace-dev-*` or `optitrace-staging-*` project), `GCP_WORKLOAD_IDENTITY_PROVIDER` and `GCP_RELEASE_SERVICE_ACCOUNT`.
3. Create a restricted encrypted remote state bucket through the approved bootstrap process. OAuth/SQL secrets are sensitive values in Terraform state and saved plans. Never publish plans as public artifacts.
4. Copy `terraform.tfvars.example` into an ignored private variable file and supply secret inputs securely. Set approved image digests and origin. Initialize the GCS backend with its explicit bucket/prefix, then create and review a Terraform plan. Infrastructure creation requires a privileged operator separate from the restricted release principal.
5. Configure verified `operational_notification_channels` separately from billing recipients; production validation rejects no operational recipient. Complete `infra/cost-rates.example.json` from current Johannesburg pricing and declared workload assumptions. Run `scripts/Estimate-Cost.ps1`. SQL runs continuously; include HA pricing for production. Budget alerts do not cap spend.
6. Apply only the reviewed plan after approval. First bootstrap registry/state/identity prerequisites through a separately reviewed infrastructure stage if image publication requires the registry before the full stack exists. Do not use an unreviewed targeted apply as a shortcut.

## Database and traffic

Take/verify a database backup before changes. Run the migration Cloud Run Job with the migration principal and wait for success; applications never migrate on startup. Use expand/contract changes compatible with the previous application revision. Run the initial administrative bootstrap without synthetic fixtures.

Verify application sign-in, membership denial, independent policy approval, evaluation trace persistence, queue/lease recovery, authenticated report export and provider disable handling. The private simulator is invoked with audience-bound service identity; its mock identity endpoints are absent in hosted mode.

The manual `Stage reviewed OptiTrace staging candidate` workflow is hardcoded to the protected staging environment and only accepts its explicitly approved isolated OptiTrace dev/staging project. Every cloud command passes that project; the default gcloud project is never changed. The workflow cannot select production. It executes compatible migrations and stages tagged API/web revisions with no traffic. Its public page/process-health probes are recorded as **process health only**, with authenticated acceptance `not_run`; it cannot promote traffic and leaves serving benchmark/simulator images unchanged. Migrations still affect the serving database and require release approval and backward compatibility.

Before promotion, complete the real-provider and paired authenticated journeys on the approved stable staging origin. A tagged web page can still call the old API and redirect Google callbacks to the stable origin; see GOOGLE-ACCEPTANCE.md. Verify worker/simulator compatibility and archive their previous configurations/images as well as API/web revisions. No candidate job can be assumed isolated: updating a Cloud Run Job changes future executions immediately.

A second explicit release review must approve the exact commit, image digests, migration execution, paired authenticated acceptance, controlled performance, latest vulnerability findings, restore rehearsal and verified alert delivery. If any are absent or a correctness/tenant-isolation issue remains, hold promotion. Update worker and simulator to approved compatible images only as part of that coordinated release, then promote the accepted API/web revisions gradually using explicit revision names. Do not use `latest` as a promotion target. There is deliberately no automatic promotion job until this evidence gate is implemented.

After approval an operator can use the following PowerShell command for each service/revision; a production promotion is not authorized by copying it:

```powershell
gcloud run services update-traffic '<approved-service>' --to-revisions='<approved-candidate-revision>=5,<recorded-stable-revision>=95' --project='<approved-project>' --region=africa-south1
if ($LASTEXITCODE -ne 0) { throw 'Traffic change failed; inspect actual traffic before retrying.' }
# Increase only after the acceptance window and an explicitly reviewed traffic decision.
```

Use at least 15 minutes of post-change monitoring and sufficient requests to interpret p95; a quiet window does not establish performance. Roll back immediately for authorization/isolation/correctness failures. Investigate/roll back sustained API 5xx >1% over five minutes or request p95 >500ms over five minutes, considering cold starts separately. A >10% controlled warm p95 regression against the strongest eligible baseline blocks acceptance even when operational alarms are quiet. Confirm expected benchmark jobs complete/recover and have no expired unclaimed leases; the current telemetry has no dedicated stuck-queue-age alert, so this remains a release evidence requirement before production.

Measure cold-start and warm end-to-end latency separately; keep engine-only synthetic results labelled. Monitor errors, database health, failed/stuck jobs and p95 latency. Staging is zonal; production SQL is regional HA.

## Rollback and restore

Record previous revision names and image digests before promotion. On authentication/isolation/correctness failures or sustained latency/error regression, route traffic back to the previous compatible revision and pause new benchmark launches. Never undo an additive migration blindly. If data repair is required, restore a backup/PITR to a new instance, validate row counts/tenant isolation, then switch the approved connection secret and revisions.

Rehearse backup restore on a disposable instance before production acceptance. A configured backup policy alone is not proof of restore capability. Keep deletion protection enabled for cloud resources and preserve evidence from failed attempts.

## Required release evidence

- Exact candidate commit and successful hosted CI run for that commit; protected reviewers and WIF identities checked.
- Reviewed private Terraform plan and resource/cost inventory; SQL 18 regional support/quotas checked in the approved project.
- Immutable digest/provenance/scans for all four runtime components; previous revisions, traffic splits and worker/simulator configs archived privately.
- Compatible migration execution with synthetic cross-tenant/RLS tests under the non-owner application login; application refuses elevated roles.
- Authenticated Google/session/membership/CSRF/role/trace/export/job acceptance with pass/fail/not-run per case.
- Backup/PITR restored into a new disposable instance, with recovered schema/data integrity, tenant boundaries, migration compatibility and measured RTO/RPO; restore target/temporary costs explicitly approved.
- Verified alert recipient delivery for API errors/request latency, worker failures and database saturation; a test notification alone is not an end-to-end incident drill. Stuck-queue monitoring gap resolved or an approved operational control demonstrated before production.
- Explicit rollback targets and decision owner. To roll back, use `update-traffic --to-revisions=<recorded-revision>=100` for each compatible service and restore recorded worker/simulator configuration. Keep migration history; do not issue a database downgrade by default.

[Cloud Run revision traffic](https://docs.cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration), [Cloud SQL restore/PITR](https://docs.cloud.google.com/sql/docs/postgres/backup-recovery/pitr), [WIF pipeline guidance](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines) and [Cloud Run monitoring](https://docs.cloud.google.com/run/docs/monitoring) were consulted on 2026-10-23. No cloud behavior, restore, IAM access, alert delivery or hosted CI success is claimed from offline validation.
