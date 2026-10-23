# Isolated OptiTrace cloud bootstrap proposal

Prepared 2026-10-23; **proposal only, no cloud writes executed**. CLI access is available, but there is no OptiTrace project. Read-only discovery showed one visible organization and one billing account with `open=False`. Keep their IDs in the private approval packet. Reopen/select an open billing account first; no paid setup can proceed against a closed account.

## Reviewable scope

Approve two stages independently:

1. **Identity-only development:** a new isolated project with proposed ID `optitrace-dev-<unique-suffix>` and display name `OptiTrace Development`; choose the visible organization only if approved. Link the specifically approved open billing account, configure Identity Platform/Google OAuth consent and a web OAuth client with the localhost callback. Keep database, API, worker and UI local. No Cloud SQL/Cloud Run instances, DNS changes, cross-project IAM, service-account keys or production data are needed for this stage. Google user creation/provider changes are cloud writes and require this explicit approval too.
2. **Hosted staging:** after local real-Google acceptance, bootstrap private state/Artifact Registry, publish reviewed immutable images, and approve a full Terraform plan for the same isolated project with `environment=staging`, regional resources in `africa-south1`. This stage adds billable SQL/compute/storage and service identities. Production remains a separate project/release decision.

Use the existing Terraform staging topology: PostgreSQL 18 Enterprise, zonal dedicated 1 vCPU/3.75 GiB, 10 GiB SSD with backups/PITR; API/web 1 vCPU/512 MiB and simulator 1 vCPU/256 MiB, minimum instances zero; bounded maximums; worker 1 vCPU/512 MiB; IAM-private simulator, private report bucket and Secret Manager. No paid VPC connector or load balancer is proposed. Stable default Cloud Run HTTPS origin is sufficient once explicitly registered with Google; custom domain/DNS requires separate approval.

Staging missed-job recovery now runs every 15 minutes; direct submissions still launch immediately. Production keeps every-minute recovery. Staging accepts a slower recovery bound and is not a production availability configuration.

## Cost assumptions for approval

USD, 730 hours/month, on-demand with no committed-use discounts, tax/currency conversion, free-tier credits or existing-account credits assumed. These are **estimates**, not observed expenditure, MTN supplier savings or hard spend limits.

| Component | Declared estimate/assumption |
|---|---|
| Google sign-in | 100 Google/social MAU: USD 0 under the currently published first-50,000 Tier 1 MAU bracket; other providers/MFA and future tariff changes excluded. |
| Staging SQL compute | `(1 × 0.0413 + 3.75 × 0.007) × 730` = **USD 49.31/month**. |
| Staging SQL SSD/backups | 10 GiB SSD × 0.17 + assumed 10 GiB used backup × 0.08 = **USD 2.50/month**; backup/PITR growth can increase this. |
| Staging scheduled recovery | 2,920 launches/month × minimum 60 seconds × `(1 × 0.000018 + 0.5 × 0.000002)` = **USD 3.33/month**, before credits. Work beyond one minute costs more. |
| Production comparison only | HA compute **USD 98.62**, 10 GiB HA SSD plus assumed backups **USD 4.20**, every-minute recovery **USD 49.93**: **USD 152.75/month subtotal**, before services, real benchmark work and ancillary costs. Production is not proposed for this approval. |

Staging's listed subtotal is **USD 55.14/month** before real workloads, Cloud Run requests/services, actual benchmark duration, scheduler control-plane charge, Secret Manager, report/image storage, logging, networking and any restore rehearsal. Proposed **USD 80 monthly planning envelope** for a small demo is a review target, not a complete quote or spending cap. Use the Google calculator with approved load assumptions before hosted-stage approval. Start the actual rate file from `infra/cost-rates.example.json`; its missing rates deliberately prevent `scripts/Estimate-Cost.ps1` from claiming a complete estimate.

The Johannesburg SQL CPU/memory and storage figures were read from the official pricing page's region-specific data; Cloud Run lists Johannesburg in Tier 1. Jobs bill a minimum minute even when no work exists. At the previous every-minute cadence, recovery alone has 43,800 minimum-minute launches/month (USD 49.93 before credits). This is why staging recovery was reduced. Price sources checked 2026-10-23: [Cloud SQL](https://cloud.google.com/sql/pricing), [Cloud Run](https://cloud.google.com/run/pricing), [Identity Platform](https://cloud.google.com/identity-platform/pricing), [calculator](https://cloud.google.com/products/calculator). Recheck rates at approval time; do not apply Iowa rates to a region without confirming the regional tariff.

## PowerShell scaffold after approval

The following commands **create/modify cloud resources**. They are preparation, not an instruction to execute before approval. Replace every placeholder in a private shell. Never change gcloud's default project or reuse another application's project. Stop on each nonzero exit.

```powershell
$ErrorActionPreference = 'Stop'
$OptiTraceProject = '<approved-unique-optitrace-dev-project-id>'
$OptiTraceBilling = '<approved-open-billing-account-id>'
$OptiTraceOrganization = '<approved-organization-id>'
$OptiTraceRegion = 'africa-south1'
# Confirm the chosen account is open and the operator can create/link this project.
$OptiTraceBillingOpen = & gcloud billing accounts describe $OptiTraceBilling --format='value(open)'
if ($LASTEXITCODE -ne 0 -or $OptiTraceBillingOpen -ne 'True') { throw 'An explicitly approved open billing account is required.' }
gcloud projects create $OptiTraceProject --name='OptiTrace Development' --organization=$OptiTraceOrganization
if ($LASTEXITCODE -ne 0) { throw 'Project creation failed.' }
gcloud billing projects link $OptiTraceProject --billing-account=$OptiTraceBilling
if ($LASTEXITCODE -ne 0) { throw 'Billing link failed.' }
gcloud services enable identitytoolkit.googleapis.com --project=$OptiTraceProject
if ($LASTEXITCODE -ne 0) { throw 'Identity Platform API enablement failed.' }
```

Google OAuth consent/web-client setup is a Google Auth Platform console prerequisite; do not substitute `gcloud iam oauth-clients` (a different IAM facility) or invent a general web-client creation command. Configure approved test users, exact localhost redirect and Google provider. Obtain the API key/client secret through the approved private secret workflow. Validate real Google login locally using [GOOGLE-ACCEPTANCE.md](GOOGLE-ACCEPTANCE.md). Grant only the approved account-status lookup access (`firebaseauth.users.get`) to the development runtime identity; no Owner/Editor runtime role and no JSON service-account key.

Hosted-stage bootstrap, only after separate resource/cost approval:

```powershell
# Provision encrypted remote-state bucket and Artifact Registry as an explicitly
# reviewed bootstrap stage; IAM/state policy is reviewed before storing secrets.
gcloud services enable run.googleapis.com sqladmin.googleapis.com secretmanager.googleapis.com artifactregistry.googleapis.com cloudscheduler.googleapis.com iam.googleapis.com iamcredentials.googleapis.com sts.googleapis.com monitoring.googleapis.com logging.googleapis.com billingbudgets.googleapis.com --project=$OptiTraceProject
if ($LASTEXITCODE -ne 0) { throw 'Service enablement failed.' }
gcloud artifacts repositories create optitrace-staging --repository-format=docker --location=$OptiTraceRegion --project=$OptiTraceProject
if ($LASTEXITCODE -ne 0) { throw 'Registry bootstrap failed.' }
# Build/review/publish image digests after registry approval. Then import the
# bootstrap registry into the single full-stack Terraform state; do not create it twice.
# Supply approved tfvars, private secrets and backend bucket/prefix first.
terraform -chdir=infra/terraform init -backend-config='bucket=<approved-private-state-bucket>' -backend-config='prefix=optitrace/staging'
if ($LASTEXITCODE -ne 0) { throw 'Backend initialization failed.' }
terraform -chdir=infra/terraform import google_artifact_registry_repository.images "projects/$OptiTraceProject/locations/$OptiTraceRegion/repositories/optitrace-staging"
if ($LASTEXITCODE -ne 0) { throw 'Reviewed bootstrap import failed.' }
terraform -chdir=infra/terraform plan -out='<private-reviewed-plan-path>'
if ($LASTEXITCODE -ne 0) { throw 'Plan failed.' }
# STOP: approve the exact private plan, resource/cost inventory and images before apply.
```

A first deployment creates Cloud Run services that become reachable; bootstrap is a release, not a harmless dry run. Keep placeholder/empty secrets out of a real apply. Terraform state/plan contains OAuth and database secrets; local state must never become a public artifact. Budget notifications do not stop spending. Retain deletion protection and authorize cleanup/restore costs explicitly.

## Approval packet fields

Project ID/display name; organization/folder; open billing account; identity-only versus hosted stage; approved Google accounts and initial tenant administrator; exact origins/callbacks; repository URL and immutable owner/repository IDs; protected GitHub environment/reviewers; private state bucket/access policy; operational/budget notification channels; current pricing export/workload assumptions; monthly planning envelope; image digests; migration/restore/rollback evidence; and a dated approver decision. Do not place secret values in this packet.
