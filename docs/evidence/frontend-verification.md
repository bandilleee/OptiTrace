# Approved frontend verification — 22 October 2026

The approved HTML design is implemented in Next.js with actual shadcn/ui components, custom light/dark semantic tokens, local Montserrat fonts and the supplied motion/logo assets. The implementation commit is `3f79186`, based on `709d5be`. The original design and ZIP are preserved. See [design notes](../design/FRONTEND.md) and [structured handback](frontend-handback.json).

## Observed results

| Check | Result |
|---|---|
| Formatting, TypeScript, production Next build | Pass |
| API/proxy unit tests | 9 pass |
| Production-container fixture browser suite | 9 pass |
| Production-container real API browser suite | 4 pass |
| Automated WCAG A/AA browser checks | No reported violations in the playground, in both themes |
| Mobile layout | 390px viewport without horizontal overflow; navigation sheet and keyboard tabs pass |
| Persisted preferences | Theme and motion choices survive reload; Escape restores dialog-trigger focus |
| Failure presentation | Failed submission shows the API error without inventing a result |
| Real workflows | Both policy profiles, persisted trace, benchmark/export, independent author/reviewer, credentials, invitations and auditor denial pass |
| Production static assets | Checked logos and fonts return200 with SVG/font content types |
| Browser runtime | No captured page errors in the manual desktop/mobile/theme review |
| OpenAPI generation | No drift; offline cross-file check passes for26 operations |
| npm audit | Zero reported vulnerabilities |
| Rebuilt web image | Zero High/Critical findings using the cached Trivy database, with unfixed findings included |

The Docker runtime includes `public/` so branding and fonts work in standalone deployment. Its dependency installation now uses a BuildKit npm cache and bounded fetch settings. The separate npm audit remains required; the Docker install does not perform a duplicate audit request.

Initial quiet package installation was cancelled after prolonged downloads. Fetch logs isolated slow Linux compiler/CSS downloads and the Next.js tarball. The final build used a local cache warmed with the exact Next.js archive from the Windows npm cache, verified against its lockfile SHA-512 integrity. Normal locked installation and the Linux production build then completed. No runtime package substitutions or dependency integrity bypasses were used.

## Screenshots

- [Dark playground](screenshots/frontend-playground-dark.png)
- [Light playground](screenshots/frontend-playground-light.png)
- [Mobile playground](screenshots/frontend-playground-mobile.png)
- [Sign-in](screenshots/frontend-login.png)
- [Benchmark](screenshots/frontend-benchmark.png)

## Limits

The validation is local Chromium evidence. Automated accessibility checks do not replace a complete manual accessibility audit or cross-browser/device acceptance. Live Google authentication, hosted CI and GCP release acceptance remain separate work.

An initial run against the long-running existing API/worker returned policy-evaluator-failure traces and benchmark disagreement. Restarting those local services cleared the repeated acceptance run. No Go code or verification rules changed in this design task; the restart is not presented as a root-cause fix or proof of long-duration engine reliability. The interface faithfully retains those indeterminate traces.

Raw logs are in ignored `.artifacts/frontend-*.log` and `.artifacts/scans/web-redesign.json`. The supplied user `artifacts/` directory is preserved outside the implementation commit. No cloud resources were changed.
