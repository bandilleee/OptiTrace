# Offline Google/GCP release preparation

Date: 2026-10-23 (Africa/Johannesburg). Input commit: `0936c8bd8e2e297993e9a104224a6d4777919a24`.

**Offline preparation: PASS. Hosted release: HOLD.** This work used an isolated managed worktree, read the engineering contract and deploy-checklist skill, changed only assigned infrastructure/CI/runbook/evidence paths, and performed no cloud resource writes or deployment. No Firebase SDK or emulator was introduced. The application/evaluator is unchanged by this task.

## Configuration and blocking prerequisites

- No OptiTrace project, OAuth client, client secret or Identity Platform API key is configured in the process environment; only an example environment file and placeholder Terraform variables were present. There is no Git remote in this checkout, so no hosted CI run/repository settings could be accepted.
- The CLI has an active account, but its default project is an unrelated application and was neither reused nor inspected. Read-only billing discovery reported the sole visible account as `open=False`; one organization is visible. This does not establish project-creator, billing-link or organization-policy permission. IDs and secret values are excluded from this tracked evidence.
- User willingness to provide CLI access is not approval of a concrete project/billing/IAM/resource plan. The reviewable [bootstrap proposal](../runbooks/CLOUD-BOOTSTRAP.md) separates identity-only local development from paid hosted staging. Reopen/select an approved open billing account and approve project identity/location/accounts before creation/configuration.
- [Google acceptance](../runbooks/GOOGLE-ACCEPTANCE.md) names actual Go configuration and REST endpoints, exact callbacks, ADC/account-status permission, bootstrap invitation, negative cases and redacted evidence requirements. It explicitly distinguishes protocol-double checks from real-provider results and tagged process probes from paired authenticated acceptance.

## Changes

- Terraform WIF binds immutable numeric GitHub repository/owner IDs as well as repository name, main and the protected environment; federation membership uses the repository ID. Example variables and mocks include the new required inputs. This follows Google's namespace-reuse guidance.
- Hosted origin validation rejects credentials, query/fragment/path, ports and non-DNS origins. The migration job now receives `APP_ENV=production`, preserving its built-in synthetic-seed rejection.
- Operational Monitoring recipients are independent of billing recipients; production rejects an empty operational recipient list. The API latency alarm is accurately labelled request latency, including cold requests.
- Missed-job recovery runs every 15 minutes in staging and every minute in production. Direct benchmark submission still launches immediately. The slower staging recovery bound is documented.
- The release workflow is explicitly **stage-only**: compatible migrations, no-traffic API/web revisions and process probes. It no longer permits 100% traffic promotion or updates serving worker/simulator images on the strength of public probes. Evidence says authenticated acceptance `not_run` and promoted `false`. A separately reviewed promotion procedure records worker/simulator compatibility and previous revisions. Staging still mutates the shared database, and first Terraform creation makes resources reachable; approval remains mandatory.
- CI selects Terraform 1.16.1 to match the locally validated CLI. Provider locks were reused unchanged with read-only initialization.
- `infra/costs.staging-proposal.json` and the bootstrap runbook record current official prices and explicit workload assumptions: USD 55.14/month listed staging subtotal (SQL plus scheduled recovery), before real workloads/ancillaries/credits/tax. USD 80 is a proposed small-demo planning envelope, **not a full quote or spending cap**. The example full-rate file remains incomplete/fail-closed rather than inventing missing prices. No observed cloud spend or MTN savings is claimed.

## Validation

All final checks exited 0; raw output is [release-next-checks.log](release-next-checks.log).

| Check | Result |
|---|---|
| Terraform init, backend disabled, locked providers read-only | PASS; no remote state or cloud credential operation |
| Terraform recursive format and validate | PASS |
| Terraform mock plans | 7/7 PASS: staging/production, malformed origin, name-as-ID rejection, missing production recipient/empty resource rejection, immutable WIF claims |
| actionlint 1.7.7 on CI and release | PASS; optional shellcheck/pyflakes integrations disabled, not claimed |
| Docker Compose config | PASS; no containers started/restarted |
| Go identity protocol suite | PASS; issuer/audience/expiry/PKCE/nonce/replay/disabled-account fixtures and hosted mock rejection, not live Google |
| PowerShell AST | Six embedded release/bootstrap snippets parse; no embedded cloud command executed |
| Git whitespace check | PASS |

An early mock assertion depended on an unknown planned resource member; it was corrected to assert the known WIF condition, and the final seven mock runs all pass. PyYAML was unavailable; actionlint supplied actual workflow parsing/expression validation instead. No missing optional tool is represented as a successful check.

## Release gaps that remain

Actual Google OAuth/provider behavior, Cloud SQL SQL-role hardening compatibility, regional quota/version acceptance, Cloud Run private invocation, real WIF/GitHub protection, registry provenance, authenticated paired release, migration against managed SQL, restore/PITR drill, operational notification delivery and cloud performance have not been exercised. A dedicated stuck-queue-age alert is not yet implemented; production requires that gap resolved or a demonstrated approved operational control. Promotion automation remains intentionally unavailable until its authenticated evidence gate is implemented. Existing local evaluator reliability/performance issues are owned by the parallel engine work, not accepted by this infrastructure report.

Sources checked 2026-10-23: [Identity Platform REST](https://docs.cloud.google.com/identity-platform/docs/use-rest-api), [credential exchange](https://docs.cloud.google.com/identity-platform/docs/reference/rest/v1/accounts/signInWithIdp), [WIF pipeline guidance](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines), [Cloud Run traffic](https://docs.cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration), [Cloud SQL PITR](https://docs.cloud.google.com/sql/docs/postgres/backup-recovery/pitr), [Cloud SQL region availability](https://docs.cloud.google.com/sql/docs/postgres/region-availability-overview), [Cloud SQL pricing](https://cloud.google.com/sql/pricing), [Cloud Run pricing](https://cloud.google.com/run/pricing), [Identity Platform pricing](https://cloud.google.com/identity-platform/pricing), and [billing prerequisites](https://docs.cloud.google.com/billing/docs/how-to/modify-project).
