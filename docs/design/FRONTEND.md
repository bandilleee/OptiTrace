# Approved design implementation

The user-approved HTML and logo kit are the visual source for this Next.js implementation. See `source-provenance.json` for source paths and hashes. The original files and Context/ are preserved.

## Visual system

Montserrat is served from local font files at weights 400, 500, 600 and 700. The desktop shell uses a 236px sidebar, 75px header, restrained borders and a main/context-column playground layout. At smaller widths the context panels stack and navigation moves into a keyboard-accessible sheet.

| Semantic token | Light | Dark |
|---|---|---|
| Background | `#ffffff` | `#000000` |
| Card | `#ffffff` | `#090909` |
| Foreground | `#202735` | `#f5f5f7` |
| Primary action | `#be3455` | `#ff8eaa` |
| Selection | `#be3455` | `#6ae0ef` |
| Border | `#e4e7ec` | `#303034` |

`apps/web/components.json` configures shadcn/ui with CSS variables and Lucide. Tailwind4 semantic colors, sidebar colors, charts, focus rings and radii map to the approved palette in `src/app/globals.css`; the installed components use those tokens. This follows the [shadcn theme-variable convention](https://ui.shadcn.com/docs/theming).

Real shadcn components provide Button, Card, Badge, Input, Label, Select, RadioGroup, Table, Tabs, Dialog, Sheet, Tooltip, Alert and Skeleton. Radix Select supplies branded dropdowns with keyboard navigation, typeahead and focus restoration. Preferences persist locally, while the supplied SVGs honor reduced motion. Explicit pause switches to the static asset. No animation framework is required.

## Product adaptation

- Sign-in uses the real existing Go-managed login flow. The prototype identity selector and invented identity are removed.
- Policy cards and the version selector contain active policies from the authenticated tenant. The Go approval and authorization boundaries are unchanged.
- Scenario and strategy choices match the existing API contract, including all five execution strategies. No browser evaluator computes an outcome.
- Results, history, evidence, timestamps, charges and exports come from actual API responses. Selecting an earlier result does not silently change the next-run inputs; an explicit notice distinguishes them.
- Configured charges are read-only trace values. The prototype price editor and fictional prices are removed.
- Freshness remains60 seconds, with a24-hour change window. Assumptions are documented in the interface rather than adopting the template's conflicting freshness example.
- Valid, adverse, unknown, stale, unavailable and skipped states remain distinct. Unconfirmed aggregate cost and call counts are never displayed as zero.
- Selecting a benchmark opens a scrollable comparison dialog, with loading, retry, completion, export and worker-failure states. Closing restores focus and stops detail polling; the run list refreshes while mounted. Benchmarks poll durable jobs. Charts compare a single profile and measurement type at a time; complete table/export evidence remains available. They do not claim commercial savings.
- Administration uses separate members, invitations and credential tabs. Auditor restrictions and independent policy approval still use the existing backend enforcement.
- Search filters the currently loaded records; it is not represented as an exhaustive database search.

## Validation

From the repository root in PowerShell:

```powershell
./scripts/Check.ps1 -Scope Web
./scripts/Generate-Client.ps1 -Check

$env:PLAYWRIGHT_BASE_URL = 'http://localhost:3000'
npm --prefix apps/web run test:e2e
$env:OPTITRACE_LIVE = '1'
npm --prefix apps/web run test:e2e
Remove-Item Env:OPTITRACE_LIVE
```

The fixture browser suite includes both-theme automated WCAG A/AA checks, keyboard tabs, mobile drawer operation, persisted preferences and failure states. The separate live suite exercises actual login, both policy profiles, trace persistence, benchmarks, independent approval, credentials, invitations and auditor denial. Automated accessibility checks are bounded evidence, not a complete manual accessibility certification.

Production packaging copies both standalone Next output and `public/` fonts/logos into the non-root runtime. No cloud deployment or live Google acceptance is part of this design change.

## Workspace roles and form behaviour

The initial overview and visible navigation adapt to the authenticated membership. People with multiple roles see their combined workflows; no role selector impersonates another identity. UI visibility is convenience only: Go and PostgreSQL remain authoritative on every request.

| Workspace role | Dashboard task | Visible workflows |
|---|---|---|
| Tenant administrator | Manage access | Overview, Administration |
| Policy author | Write and submit | Overview, Policy library |
| Policy reviewer | Independently approve and activate | Overview, Policy library |
| Integration developer | Run and measure | Overview, Playground, Execution traces, Policy library, Benchmarks |
| Auditor | Inspect evidence | Overview, Execution traces, Policy library, Benchmarks, Audit history |

Platform operators use the separately authorized provisioning workflow and do not inherit tenant evidence access. Service principals use scoped credentials, not browser dashboards.

Policy and credential names are trimmed, reject blanks/control characters and observe the API's 120-byte UTF-8 limit. Invitations accept conventional Google account email addresses without display names, with syntax and length validation in the browser and Go. Roles/scopes come from explicit choices; empty role sets and duplicate API scopes are rejected. Seeds must be JavaScript-safe integers and counts must be whole numbers from 2 through 1000. Go independently enforces its signed-int64 and count boundaries. Resource schemas and authorization are unchanged.

Field errors appear beside inputs with accessible descriptions. Successful saves are announced in the app. Failed role saves preserve the selection for retry. Credential revocation requires a named confirmation dialog; cancelling it makes no request. No normal tenant action requires a terminal, developer console, native alert, prompt or confirm box. Deployment and platform provisioning remain separate operational tasks.
