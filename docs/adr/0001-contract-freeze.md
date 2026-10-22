# ADR-0001: Freeze bounded policies and server-owned authorization

**Status:** Accepted by integrator after contract regeneration and policy guard repair
**Date:** 2026-10-22
**Deciders:** OptiTrace integrator and independent policy reviewer
**Input revision:** b39abee7dbd3258fc793050be773305827dbbac0

## Context

The existing console, Go engine, PostgreSQL schema and synthetic benchmark need one shared interpretation of policy outcomes, tenant boundaries and wire behavior. An optimizer must not change policy correctness, omit compulsory work or hide charges. Documentation and an API schema can define requirements but cannot certify runtime enforcement. Context/ is preserved.

## Decision

Freeze [policy semantics](../../contracts/POLICY.md), [authorization/API behavior](../../contracts/AUTHORIZATION.md) and [OpenAPI](../../contracts/openapi.json) against the input revision. Keep existing routes, fields, enums and Go interface; make compatible schema corrections for existing Go responses and authorization. New optional observation fields describe existing output. Keep reserved response enum values; do not infer an operation from an enum.

Use the bounded typed profiles and generated embedded OPA. The completion oracle, not a cost estimate or LLM, controls stopping. Go resolves principals/membership and applies roles; PostgreSQL provides tenant isolation, immutable evidence and approval guards. Next is a same-origin presentation/proxy layer. Separate durable reservations/journals from final immutable evaluations. Retain uncertainty when process failure prevents confirmed billing.

Durable leased PostgreSQL benchmark jobs avoid a second queue in this bounded release. Compare identical immutable synthetic observations, separating tuning from held-out evaluation. Reports retain adverse/negative findings. Full authenticated API latency, synthetic cost savings and measured engine latency have distinct acceptance gates.

## Options considered

| Option | Complexity | Correctness/isolation | Scale/cost consequences |
|---|---|---|---|
| Bounded profiles + Go + PostgreSQL + embedded OPA (selected) | Bounded proof and one authoritative database | Exhaustive three-capability completion checks and defense in depth | Adequate bounded jobs; synthetic prices only |
| Arbitrary executable policy/LLM decisions | Unbounded rule and review surface | No equivalent bounded completion proof | Flexible but requires a separate contract and validation programme |
| Browser-only tenant/role checks | Low initial implementation effort | Direct API access bypasses checks | Unacceptable authorization boundary |
| Ephemeral background benchmark goroutine | Simple scheduling | Process loss loses work/ownership | Does not satisfy durable job requirement |

## Trade-offs and consequences

Three capabilities and fixed windows permit exhaustive proof but do not generalize to arbitrary policies. Tenant RLS limits data exposure but does not implement end-user roles. Journaling narrows the crash window but cannot prove whether a supplier charged a request after process loss; reconciliation must retain unknown cost. Lease retries need fencing and must not be confused with permission to replay billable source calls.

Cloud Run/Cloud SQL/Identity Platform remains the existing hosting direction. This freeze performs no deployment, price lookup or hosted validation and grants no release approval. New semantics require a new version; archived evaluations must retain their original meaning across deployments.

## Action items

- [x] Integrator reviews shared contract diff and regenerates frontend OpenAPI types; architect cannot edit that path.
- [x] Run disposable PostgreSQL authorization/invariant tests and Linux race checks in an authorized environment.
- [x] Review database guards against all direct UPDATE/DELETE paths, including kind/status changes; Go checks alone do not establish database immutability.
- [ ] Validate configured Google Identity Platform login and release gates before hosted acceptance.
- [ ] Define supplier reconciliation and explicit semantic artifact pinning before expanding beyond fixed synthetic-v1 policies.

## Evidence

See [architect review](../evidence/architect-review.md) for observed commands, source mapping and limitations. This ADR does not mark those action items complete.

Integrator follow-up: contract types and SDK regenerated; migration2 and runtime-role PostgreSQL tests close the policy guard finding. Original architect evidence remains a dated review snapshot. Hosted identity and supplier reconciliation remain open.
