# Contributing

Start with the README's local setup and `AGENTS.md`. Issues and pull requests are
welcome. Describe the problem, changes, validation commands and limitations.

Use synthetic data only. Keep tenant authorization in Go/PostgreSQL, preserve
approved-policy immutability and the author/reviewer boundary, charge every started
billable call, and distinguish unavailable/stale/adverse evidence. No runtime LLM
decides verification outcomes. Never commit real credentials, customer data or
private deployment files.

Run relevant Go and web checks plus generated-contract drift checks for API changes.
Database/race/browser checks are described in the README. Changes to policy or
planner behavior need correctness evidence before efficiency comparisons. Do not
report simulated supplier charges as measured network-provider savings.

Code contributions are licensed under MIT. Bundled third-party assets retain their
own licenses; see `THIRD_PARTY_NOTICES.md`.
