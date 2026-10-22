# Release and rollback

Cloud writes require explicit release approval. No project, OAuth client, registry or database is created by running local development/tests.

## Prerequisites and identity

Select an existing approved Google Cloud project and billing account. Create a dedicated development project for optional real Google sign-in tests. Configure Google OAuth consent and exact redirects: `http://localhost:3000/api/v1/auth/callback` locally and `<approved HTTPS origin>/api/v1/auth/callback` in hosting. Enable the Google provider in Identity Platform. Keep client secrets in the approved secrets workflow. Invite the initial operator/admin via the privileged administrative CLI; never enable the synthetic seed in hosting.

Application tenants are PostgreSQL entities; v1 does not use Identity Platform tenants. Google and Identity Platform account/linking errors fail sign-in rather than silently linking accounts. Verify the real provider path on a dedicated project before accepting a cloud release.

## Infrastructure review

1. Run all local/CI checks and independent security acceptance against the candidate commit. Review dependency and image findings.
2. Build/publish immutable image digests using a protected release workflow and Workload Identity Federation. Do not use service-account JSON keys. The Terraform trust condition binds the exact repository, main branch and protected GitHub environment.
3. Create a restricted encrypted remote state bucket through the approved bootstrap process. OAuth/SQL secrets are sensitive values in Terraform state and saved plans. Never publish plans as public artifacts.
4. Copy `terraform.tfvars.example` into an ignored private variable file and supply secret inputs securely. Set approved image digests and origin. Initialize the GCS backend with its explicit bucket/prefix, then create and review a Terraform plan. Infrastructure creation requires a privileged operator separate from the restricted release principal.
5. Complete `infra/cost-rates.example.json` from current Johannesburg pricing and declared workload assumptions. Run `scripts/Estimate-Cost.ps1`. SQL runs continuously; include HA pricing for production. Budget alerts do not cap spend.
6. Apply only the reviewed plan after approval. First bootstrap registry/state/identity prerequisites through a separately reviewed infrastructure stage if image publication requires the registry before the full stack exists. Do not use an unreviewed targeted apply as a shortcut.

## Database and traffic

Take/verify a database backup before changes. Run the migration Cloud Run Job with the migration principal and wait for success; applications never migrate on startup. Use expand/contract changes compatible with the previous application revision. Run the initial administrative bootstrap without synthetic fixtures.

Verify application sign-in, membership denial, independent policy approval, evaluation trace persistence, queue/lease recovery, authenticated report export and provider disable handling. The private simulator is invoked with audience-bound service identity; its mock identity endpoints are absent in hosted mode.

Promote the candidate Cloud Run revision gradually after smoke acceptance. Measure cold-start and warm end-to-end latency separately; keep engine-only synthetic results labelled. Monitor errors, database health, failed/stuck jobs and p95 latency. Staging is zonal; production SQL is regional HA.

## Rollback and restore

Record previous revision names and image digests before promotion. On authentication/isolation/correctness failures or sustained latency/error regression, route traffic back to the previous compatible revision and pause new benchmark launches. Never undo an additive migration blindly. If data repair is required, restore a backup/PITR to a new instance, validate row counts/tenant isolation, then switch the approved connection secret and revisions.

Rehearse backup restore on a disposable instance before production acceptance. A configured backup policy alone is not proof of restore capability. Keep deletion protection enabled for cloud resources and preserve evidence from failed attempts.
