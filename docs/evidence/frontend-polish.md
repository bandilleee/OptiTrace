# Frontend workflow polish — 22 October 2026

Input revision: `01f3a5ae73e461763bd2d3d4725cb71e7fa69ccf`.

## Delivered behaviour

- All rendered selection controls use branded shadcn/Radix dropdowns with keyboard navigation, typeahead, focus rings and the approved light/dark tokens.
- Benchmark run clicks open a focused comparison dialog instead of appending a report below the list. Launching a run opens the same dialog immediately. Loading, polling, worker errors, retry, labelled exports and complete evidence remain available. Escape closes and restores focus. Closing stops detail polling; the list refreshes separately.
- The initial overview and navigation follow actual membership roles. Administrators manage access; authors create/submit; reviewers independently approve/activate; developers execute/measure; auditors inspect evidence. Multiple roles combine. No tenant-admin superuser privilege or role impersonation was introduced. Platform operators and service principals retain their separate access models.
- Names, invitation email syntax, numeric seeds/counts and role selections receive inline validation. Go independently validates names/emails and rejects duplicate credential scopes. Required role/profile/scope enum enforcement and backend numeric limits remain authoritative.
- Successful policy and administration operations announce results in the app. Failed role saves retain selected roles for retry. Revocation has a named confirmation and cancellation path. Browser-console commands and native alert/prompt/confirm dialogs are unnecessary for tenant workflows; structured server operational logs remain intact.
- Missing benchmark cost/latency/agreement values remain unavailable rather than being represented as zero.

See [frontend design and role matrix](../design/FRONTEND.md) for the exact workflows and validation boundaries.

## Verification

| Check | Observed result |
|---|---|
| Frontend unit tests | 13 pass, including byte limits, email syntax, numeric bounds and role unions |
| TypeScript and Prettier | Exit 0 |
| Production image build | Exit 0; Next.js production build and types complete |
| Production fixture browser suite | 12 pass, including report polling/reopen/focus, narrow layout, dropdown keyboard controls, role navigation, invalid forms and failed-save retry |
| Automated accessibility | Zero WCAG 2 A/AA and 2.1 A/AA findings in both playground themes and the open report dialog |
| Real local-stack browser suite | 4 pass: Google-protocol test-double login, both policy profiles, persisted trace, worker/report/export, separate author/reviewer, credential/invitation flows and auditor 403 |
| Go suite with disposable PostgreSQL18 | Exit 0; new malformed-input HTTP checks and Unicode/email boundaries included; database tests executed |
| Go vet | Exit 0 |
| Linux validator race check | Exit 0; new validation helpers/tests execute with race detection in a Linux container; full server race suite not rerun |
| Generated OpenAPI/SDK drift | Exit 0; no shared contract or generated-client changes |
| npm audit | Zero reported vulnerabilities |
| Source inspection | No browser console calls or native alert/confirm/prompt flows in application source |
| Visual inspection | Desktop overview, desktop report and 390px mobile popup reviewed; card action spacing corrected and packaged again |

Fixture and live browser checks ran against the production Docker frontend on port3000. Browser fixtures are explicitly intercepted synthetic API tests; the four live journeys use actual local Go/PostgreSQL/simulator services and no API interception. All records are synthetic.

Final web image: `sha256:d316b2f8ea8102903e0229b166ad8a895fb7697a269ba23ac6f190445e8e3737`.
Engine image: `sha256:49c81848bcc0682856b7299bd6ec2f8516a8adf466fde422c4e2039c60bb8a77`.

## Evidence and limits

Commands, exit codes and changed paths are recorded in [frontend-polish-handback.json](frontend-polish-handback.json). Durable run outputs are preserved beside this report. Raw Playwright traces/results remain under the ignored `.artifacts/polish-*-results` directories.

Screenshots: [role overview](screenshots/polish-overview.png), [comparison dialog](screenshots/polish-benchmark.png), [mobile popup](screenshots/polish-benchmark-mobile.png).

No cloud resource was changed and no hosted Google login, hosted CI, full accessibility certification or commercial savings claim is made. Invitations still authorize sign-in without sending email. Email validation intentionally accepts conventional Google account addresses without display names. Frontend search still filters loaded records only.

The historical long-running evaluator anomaly documented in [the previous verification](frontend-verification.md) remains an unresolved reliability follow-up; this polish did not change the evaluator. The current local acceptance runs passed following the normal service rebuild/recreation. Full Linux race/performance/security acceptance from the earlier integration is historical, not rerun or claimed as fresh evidence for this turn. Cloud release remains subject to its separate acceptance and approval gates.
