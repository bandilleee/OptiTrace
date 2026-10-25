# OptiTrace architecture decisions

OptiTrace is an execution experiment over an approved verification contract, not a fraud model. The bounded first release uses synthetic South African scenarios. MTN capability availability, tariffs and savings remain unverified.

## Decisions

- Next.js App Router/TypeScript console, same-origin API proxy, and Go REST API with generated OpenAPI types. Go owns authorization and policy behaviour.
- Selected portfolio identity: Supabase Auth Google authorization-code/PKCE sign-in, verified server-to-server, with transient Google credentials checked against the linked Google subject. Go manages opaque eight-hour sessions hashed in PostgreSQL. Invitations grant membership; an upstream identity account alone never grants access. Provider tokens are not persisted. Google Cloud Identity Platform REST remains optional for the legacy GCP profile. No Firebase SDK/emulator.
- PostgreSQL18 is the only authoritative database. Application login has no superuser/BYPASSRLS privilege. Tenant context is transaction-local. Narrow security-definer functions support the few legitimate cross-tenant authentication and worker operations.
- Typed bounded policies generate embedded OPA rules. Immutable policy versions require an independent reviewer. Unknown evidence is explicit. Statistical estimates rank valid fetch plans and cannot rewrite business rules.
- Complete policy retrieves all capabilities. Conditional policy may stop optional work only after all possible completions have the same outcome and mandatory obligations are satisfied. No cross-request cache in v1.
- Benchmarks persist before launch and execute in a leased worker. Retries cannot overwrite completed reports. Same immutable observations and fault distributions feed every comparator. Actual HTTP execution is labelled separately from virtual simulation.
- Selected portfolio hosting: Vercel Next.js, one Render Free process for Go API/private simulator/lease worker, and Supabase PostgreSQL/Auth. Dedicated database role, verified TLS session pooling and explicit Data API denials protect application data. Free-tier sleep/pause and cold starts are accepted demo availability limits. See ADR0004 and the portfolio hosting runbook. GCP Terraform/Cloud Run/Cloud SQL remain optional; their Johannesburg default does not establish residency for this profile.
- Reviewed integration and explicit release approval precede cloud writes. Coding agents have no deployment credentials.

## Data flow

The browser obtains a session through the Go-managed OIDC transaction. The Next.js proxy forwards a constrained set of headers. Each tenant request checks current membership or scoped API credentials, establishes tenant context, resolves an immutable policy, fetches permitted observations, applies the evaluator and persists evidence before responding. CSRF and exact Origin checks protect cookie-authenticated mutations.

Evaluation idempotency keys bind the request hash to the persisted response. Reusing a key with a different request is rejected. Audit events capture changes without storing secrets. Benchmark manifests record source observation hashes, seeds, versions, profile/scenario allocation, prices, runtime assumptions, latency labels and hardware metadata.

## Release boundaries

The research target is 20% lower variable execution cost than the strongest eligible staged baseline with no correctness differences, no compulsory violations, warm API p95 <=500ms and <=10% relative p95 regression. An engine-only benchmark does not certify the authenticated API or cloud latency. Negative findings remain visible.

Live network integrations, customer data, learned planning, payment collection and real pilots require separate validation. Production-oriented implementation is not evidence of production acceptance or MTN commercial benefit.
