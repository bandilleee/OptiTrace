# Architect contract review evidence

Date: 2026-10-22. Input and inspected HEAD: `b39abee7dbd3258fc793050be773305827dbbac0`.
Task result: contract freeze prepared for integrator review. Runtime/release acceptance remains open. No application implementation, harness script, Context/ content, cloud resource or Git history was changed.

## Requirements reviewed

| Source | Requirement carried into freeze |
|---|---|
| Context/OptiTrace.pdf, pp. 1-3 | Bounded account verification, approved rules independent of planning, required evidence and explicit fallback, no observed MTN savings claim |
| Context/OptiTrace.pdf, pp. 3-4 | Preserve unavailable/stale states, compulsory checks and tenant/purpose/freshness boundaries; learned planning optional and deferred |
| Context/OptiTrace.pdf, pp. 5-7 | Strong staged and established policy-engine comparators, identical observations, held-out data, planner overhead and failure cases; synthetic agreement is not fraud accuracy |
| Context/OptiTrace.pdf, pp. 7-8 | Product-owner workload/latency agreement, actual billing and contribution evidence before pilot; no production readiness claim |
| AGENTS.md and contracts/IMPLEMENTATION.md | Go/PostgreSQL authorization, independent review, immutable approved versions, all started billable calls charged, configured Identity Platform and synthetic-data boundaries |
| policies/README.md and policies/synthetic-v1.json | Two fixed profiles, inclusive 60s/24h windows, three capabilities, explicit compulsory failure priority |
| services/engine/internal/evaluation/{types,policy,engine,benchmark}.go | Existing wire fields, deterministic outcome/stopping logic, journal uncertainty and labelled benchmark execution |
| services/engine/internal/server/{server,resources,admin}.go | Actual session/key matrix, policy transitions, request defaults, idempotency, membership response and API status codes |
| services/engine/internal/storage/{schema.sql,roles.go} | Transaction/RLS and privilege boundaries; static SQL gaps below |

The eight-page PDF was successfully extracted read-only with bundled Python/pypdf (exit 0), after system Python failed because pypdf was absent (exit 1). Text was reviewed for requirements; visual layout and external factual claims were not audited. No external claims/prices from the proposal were adopted as verified facts.

## Compatible contract corrections

- OpenAPI session-only operations no longer advertise bearer-key access. Evaluation endpoints retain both schemes, with scope restrictions documented.
- Cookie mutations document required CSRF and Origin headers; evaluation bearer mutations retain the exemption. Existing request/response enums are preserved.
- Optional Check event_time, subject and scope now match Go output accepted by the otherwise closed response schema.
- Policy detail GET already handled by Go is documented. Membership PATCH now describes the actual `{id,roles}` result instead of requiring an absent email.
- Tenant role input excludes the operator CLI role and describes existing unique-role bounds. The general Role response enum remains unchanged.
- Benchmark seed and zero-count defaults describe the existing handler, including nonzero signed int64 seeds. Evaluation replay, unconfirmed journal and null-cost semantics are explicit.
- POLICY.md and AUTHORIZATION.md freeze behavioral interpretation; ADR-0001 records alternatives, consequences and incomplete acceptance work.

No frontend client was regenerated because apps/web is outside writable ownership. The integrator must regenerate and review it before merging the shared contract, otherwise the existing generation-drift CI check will fail. The added contract checker is an offline cross-file consistency check, not an application or harness modification.

## Static implementation findings requiring integrator/backend follow-up

1. **Approved-policy database guard is incomplete (acceptance blocker).** schema.sql:46-55 installs policy approval guards on UPDATE only. An approved row can be changed to draft with otherwise identical data because status is excluded from the content comparison and only active status is protected against reverse transition. A subsequent update can then alter content. The guard also does not compare resource kind or install an INSERT approval guard, allowing direct database paths that the Go API does not expose. Do not describe database immutability as verified. Add database-level transition/kind/insert constraints and disposable PostgreSQL regression evidence under the runtime role before claiming this requirement complete.
2. **Fixed semantics are not independently pinned per stored version.** Stored policy metadata carries profile/version, while fixed thresholds/generated logic live in the engine artifact. A deployment changing the same profile's meaning would change existing policy behavior. Keep synthetic-v1 semantics fixed; future semantics require an explicit artifact/version binding and migration design, not an unreviewed engine update.
3. **Benchmark proof is bounded.** The implemented `opa` comparator uses staged fetching and generated embedded OPA final evaluation; it is not proof against every optimized OPA deployment. The current report selects the strongest eligible comparator across its implemented strategies, potentially full_parallel. Preserve that conservative baseline and disclose the comparator configuration. The proposal's representative workload and latency need product-owner agreement; current 500ms/10% gates remain prototype assumptions. The observed unit-test report said the synthetic research target was not established, not that MTN savings were achieved.
4. **Crash billing remains uncertain by design.** A journal starting event precedes network dispatch. It cannot prove a supplier charge after process loss. The explicit unconfirmed/null-cost path is essential; real supplier reconciliation remains unimplemented acceptance work.

These are static conclusions from the named files, not reproduced PostgreSQL exploit tests. The architecture handoff does not authorize implementation edits or release.

## Observed checks

All commands ran in Windows PowerShell. Paths below are repository-relative evidence. Native command exit codes were captured explicitly.

| Command | Observed exit | Evidence and scope |
|---|---|---|
| `python contracts/check_contract.py` | 0 | [architect-contract-check.log](architect-contract-check.log): 26 operations, local refs, path parameters, error envelopes, security/header boundaries, Go Check fields and fixture/role enums |
| PowerShell here-string piped to `python -`, compatibility assertions reproduced below | 0 | [architect-compatibility.log](architect-compatibility.log): input operations, response codes, properties, enums and required Check fields preserved |
| `go test ./... -count=1 -v` from services/engine, with `GOCACHE=Join-Path $env:TEMP 'optitrace-architect-go-cache'` | 0 | [architect-go-tests.log](architect-go-tests.log): engine exhaustive completion, OPA agreement, normalization, faults, cancellation, journaling, benchmark labels and identity tests pass; both database tests explicitly SKIP |
| `./scripts/Test-Harness.ps1` | 0 | [architect-harness.log](architect-harness.log): script parsing, task graph, handback validation and literal argument checks; existing acceptance script executed unchanged |
| `git diff --check` | 0 | [architect-diff-check.log](architect-diff-check.log); no whitespace errors (Git may emit line-ending warnings) |
| `docker version --format '{{.Server.Os}}'` | 1 | Docker config access and docker_engine named-pipe access denied; [architect-docker-check.log](architect-docker-check.log) |
| Initial `go test ./internal/evaluation ./internal/identity ./internal/storage -count=1 -v` using default GOCACHE | 1 | Default go-build cache access denied; corrected with temporary GOCACHE; subsequent targeted run exited 0 and full logged run above exited 0 |

Compatibility assertion body (run from repository root using PowerShell `@' ... '@ | python -`):

```python
import json, subprocess
from pathlib import Path
base = json.loads(subprocess.check_output([
    'git', 'show', 'b39abee7dbd3258fc793050be773305827dbbac0:contracts/openapi.json'
]))
now = json.loads(Path('contracts/openapi.json').read_text())
assert set(base['paths']) <= set(now['paths'])
for path, methods in base['paths'].items():
    assert set(methods) <= set(now['paths'][path])
    for method, op in methods.items():
        assert op['operationId'] == now['paths'][path][method]['operationId']
        assert set(op['responses']) <= set(now['paths'][path][method]['responses'])
for name, schema in base['components']['schemas'].items():
    current = now['components']['schemas'][name]
    assert set(schema.get('properties', {})) <= set(current.get('properties', {})), name
    if 'enum' in schema:
        assert schema['enum'] == current['enum'], name
assert set(base['components']['schemas']['Check']['required']) == set(
    now['components']['schemas']['Check']['required']
)
```

## Limits

No disposable PostgreSQL URL was configured. TestPostgresAuthenticatedWorkflow and TestApplicationRoleHardening skipped; no tenant/RLS, SQL immutability, concurrent idempotency or end-to-end database acceptance is claimed. Linux container race detection was blocked by Docker access; native Go tests do not substitute for it. Hosted Google Identity Platform, browser journeys, generated frontend types, cloud behavior and hosted CI were not tested. The structural checker does not replace a full OpenAPI 3.1 validator. Existing evidence files from the input revision are historical; only architect-prefixed evidence belongs to this run. Integrator review and release approval remain separate.
