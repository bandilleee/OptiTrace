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

Real shadcn components provide Button, Card, Badge, Input, Label, NativeSelect, RadioGroup, Table, Tabs, Dialog, Sheet, Tooltip, Alert and Skeleton. Native selects preserve familiar browser/keyboard operation. Preferences persist locally, while the supplied SVGs honor reduced motion. Explicit pause switches to the static asset. No animation framework is required.

## Product adaptation

- Sign-in uses the real existing Go-managed login flow. The prototype identity selector and invented identity are removed.
- Policy cards and the version selector contain active policies from the authenticated tenant. The Go approval and authorization boundaries are unchanged.
- Scenario and strategy choices match the existing API contract, including all five execution strategies. No browser evaluator computes an outcome.
- Results, history, evidence, timestamps, charges and exports come from actual API responses. Selecting an earlier result does not silently change the next-run inputs; an explicit notice distinguishes them.
- Configured charges are read-only trace values. The prototype price editor and fictional prices are removed.
- Freshness remains60 seconds, with a24-hour change window. Assumptions are documented in the interface rather than adopting the template's conflicting freshness example.
- Valid, adverse, unknown, stale, unavailable and skipped states remain distinct. Unconfirmed aggregate cost and call counts are never displayed as zero.
- Benchmarks poll durable jobs. Charts compare a single profile and measurement type at a time; complete table/export evidence remains available. They do not claim commercial savings.
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
