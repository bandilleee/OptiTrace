# Public sandbox boundary

`POST /v1/demo/evaluations` is an explicit exception to the tenant API's
authentication requirement. It is disabled unless `PUBLIC_DEMO_ENABLED=true`.
The OpenAPI operation declares `security: []`; existing tenant operations retain
their session/API-key authentication and Go/PostgreSQL authorization.

Only `profile`, `scenario` and `strategy` are accepted, from the existing bounded
enumerations. Unknown fields, tenant/policy IDs, custom policy code, phone numbers,
extra JSON values and bodies over 2048 bytes are rejected. Policies are fixed
demonstration version 1, not tenant-approved versions.

The handler invokes the real evaluator with synthetic adapters. It holds no
database handle and makes no journal, resource, audit, membership, credential or
job writes. Responses are explicitly `public_demo=true`, `persisted=false`; IDs
are ephemeral and cannot be looked up through the tenant trace API. All started
synthetic calls retain normal charge/compulsory-work semantics.

Two concurrent evaluations and 30 requests/minute are allowed per API process,
with a five-second evaluation deadline and no waiting queue. Limits are global,
not per visitor: one visitor can exhaust the shared allowance. Scaling to multiple
processes multiplies these limits. No cookies, IP addresses or external identity
are used as an authorization shortcut. The endpoint has no CORS grant. Public
synthetic results are not private and do not need session CSRF protection.

This preview is not a public benchmark service or an availability commitment.
Run a local instance for repeatable performance work. Disable the environment
flag and deploy to remove the public execution endpoint.
