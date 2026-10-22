# OptiTrace web console

Next.js App Router application with a Go-authorized, same-origin API proxy. All observations and prices are synthetic.

From this directory in PowerShell:

```powershell
npm ci
npm run api:generate
npm run dev
```

The API defaults to `http://localhost:8080`. Override `API_BASE_URL` for container networking. Open `http://localhost:3000` and continue through the configured Google or local protocol test provider. No Firebase SDK or emulator is used.

```powershell
npm run typecheck
npm test
npm run build
npx playwright install chromium
npm run test:e2e
```

Playwright UI tests deliberately use explicit API protocol stubs; they verify interaction, trace evidence, role presentation, and invitation-only entry. Backend authorization and real database/network integration are separate root acceptance gates. To run against an already started web process set `PLAYWRIGHT_BASE_URL`.

For full-stack acceptance, start the root Compose stack, then run:

```powershell
$env:OPTITRACE_LIVE = '1'
$env:PLAYWRIGHT_BASE_URL = 'http://localhost:3000'
npm run test:e2e
```

These live tests do not intercept APIs. They sign in through the local protocol provider, exercise persisted evaluations and author/reviewer transitions, create synthetic invitations and one-time credentials, and verify auditor restrictions. They require seeded demo data and leave their synthetic audit records intact.

The checked-in `src/lib/api-schema.d.ts` comes from the shared OpenAPI contract. Regenerate after approved contract changes. The proxy only targets the configured server and `/v1` paths, forwards an explicit request-header allowlist, preserves independent session cookies, and leaves provider redirects for the browser.

Production uses standalone Next output. HTTPS, identity callback allowlists, session enforcement, and tenant authorization remain mandatory deployment/backend responsibilities.
