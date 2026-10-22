# OptiTrace architecture decisions

OptiTrace is an execution experiment over an approved verification contract, not a fraud model. The bounded first release uses synthetic South African scenarios. MTN capability availability, tariffs and savings remain unverified.

## Decisions

- Next.js App Router/TypeScript console, same-origin API proxy, and Go REST API with generated OpenAPI types. Go owns authorization and policy behaviour.
- Google Cloud Identity Platform REST with Google OIDC authorization-code sign-in. Browser sessions are opaque eight-hour random credentials hashed in PostgreSQL. Invitations grant membership; an upstream identity account alone never grants access. No Firebase SDK/emulator.
- PostgreSQL18 is the only authoritative database. Application login has no superuser/BYPASSRLS privilege. Tenant context is transaction-local. Narrow security-definer functions support the few legitimate cross-tenant authentication and worker operations.
- Typed bounded policies generate embedded OPA rules. Immutable policy versions require an independent reviewer. Unknown evidence is explicit. Statistical estimates rank valid fetch plans and cannot rewrite business rules.
- Complete policy retrieves all capabilities. Conditional policy may stop optional work only after all possible completions have the same outcome and mandatory obligations are satisfied. No cross-request cache in v1.
- Benchmarks persist before launch and execute in a leased worker. Retries cannot overwrite completed reports. Same immutable observations and fault distributions feed every comparator. Actual HTTP execution is labelled separately from virtual simulation.
- Cloud Run services/jobs, Cloud SQL, Identity Platform, Secret Manager, Artifact Registry, private report bucket, Logging/Monitoring, Terraform and Workload Identity Federation. Regional compute/database defaults to Johannesburg; identity location is a separate service concern.
- Reviewed integration and explicit release approval precede cloud writes. Coding agents have no deployment credentials.

## Data flow

The browser obtains a session through the Go-managed OIDC transaction. The Next.js proxy forwards a constrained set of headers. Each tenant request checks current membership or scoped API credentials, establishes tenant context, resolves an immutable policy, fetches permitted observations, applies the evaluator and persists evidence before responding. CSRF and exact Origin checks protect cookie-authenticated mutations.

Evaluation idempotency keys bind the request hash to the persisted response. Reusing a key with a different request is rejected. Audit events capture changes without storing secrets. Benchmark manifests record source observation hashes, seeds, versions, profile/scenario allocation, prices, runtime assumptions, latency labels and hardware metadata.

## Release boundaries

The research target is 20% lower variable execution cost than the strongest eligible staged baseline with no correctness differences, no compulsory violations, warm API p95 <=500ms and <=10% relative p95 regression. An engine-only benchmark does not certify the authenticated API or cloud latency. Negative findings remain visible.

Live network integrations, customer data, learned planning, payment collection and real pilots require separate validation. Production-oriented implementation is not evidence of production acceptance or MTN commercial benefit.
