"""Offline cross-file contract checks; does not execute or certify the service.

Run from any directory with Python 3. No third-party dependencies or writes.
This is deliberately not a full OpenAPI/JSON Schema validator.
"""
import json
from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
api = json.loads((ROOT / "contracts/openapi.json").read_text())
schemas = api["components"]["schemas"]
assert api["openapi"] == "3.1.0"
assert api["components"]["securitySchemes"]["session"]["name"] == "optitrace_session"


def walk(value):
    if isinstance(value, dict):
        if "$ref" in value:
            ref = value["$ref"]
            assert ref.startswith("#/"), f"External ref needs separate review: {ref}"
            target = api
            for part in ref[2:].split("/"):
                target = target[part.replace("~1", "/").replace("~0", "~")]
        if value.get("type") == "object":
            assert set(value.get("required", [])) <= set(value.get("properties", {}))
        for child in value.values():
            walk(child)
    elif isinstance(value, list):
        for child in value:
            walk(child)


walk(api)
ids = set()
operations = 0
for path, methods in api["paths"].items():
    for method, op in methods.items():
        operations += 1
        assert op["operationId"] not in ids
        ids.add(op["operationId"])
        parameters = op.get("parameters", [])
        actual = {p["name"] for p in parameters if p["in"] == "path" and p["required"]}
        assert actual == set(re.findall(r"\{([^}]+)\}", path)), path
        public = path in {"/healthz", "/v1/auth/login", "/v1/auth/callback"}
        expected = [] if public else [{"session": []}]
        if "/evaluations" in path:
            expected.append({"integrationKey": []})
        assert op.get("security") == expected, (path, method, "security")
        if method in {"post", "patch", "delete"}:
            headers = {p["name"]: p for p in parameters if p["in"] == "header"}
            for name in ["Origin", "X-CSRF-Token"]:
                assert headers[name]["required"] == ("/evaluations" not in path)
        for status, response in op["responses"].items():
            if int(status) >= 400:
                assert response["content"]["application/json"]["schema"] == {
                    "$ref": "#/components/schemas/Error"
                }

# Cross-file checks catch drift between Go output, the frozen fixture and OpenAPI.
types = (ROOT / "services/engine/internal/evaluation/types.go").read_text()
check_body = re.search(r"type Check struct \{(.*?)\n\}", types, re.S).group(1)
go_check_fields = set(re.findall(r'json:"([^",]+)', check_body))
assert go_check_fields <= set(schemas["Check"]["properties"]), go_check_fields
fixture = json.loads((ROOT / "policies/synthetic-v1.json").read_text())
assert set(fixture["states"]) == set(schemas["Check"]["properties"]["status"]["enum"])
assert set(fixture["outcomes"]) == set(schemas["Evaluation"]["properties"]["outcome"]["enum"])
assert {p["profile"] for p in fixture["profiles"]} == set(schemas["Profile"]["enum"])
assert fixture["freshness_seconds"] == 60 and fixture["recent_hours"] == 24
for go_name, schema_name in [("Scenarios", "Scenario"), ("Strategies", "Strategy")]:
    body = re.search(r"var " + go_name + r" = \[\]string\{([^}]+)\}", types).group(1)
    assert set(re.findall(r'"([^"]+)"', body)) == set(schemas[schema_name]["enum"])
roles = (ROOT / "services/engine/internal/server/admin.go").read_text()
role_body = re.search(r"var validRoles = \[\]string\{([^}]+)\}", roles).group(1)
assert set(re.findall(r'"([^"]+)"', role_body)) == set(schemas["TenantRole"]["enum"])
for name in ["MembershipInput", "InvitationInput"]:
    assert schemas[name]["properties"]["roles"]["items"]["$ref"].endswith("/TenantRole")
assert "platform_operator" not in schemas["TenantRole"]["enum"]
assert schemas["Evaluation"]["properties"]["supplier_cost_microrand"]["type"] == ["integer", "null"]
membership_response = api["paths"]["/v1/tenants/{tenant}/memberships/{id}"]["patch"]["responses"]["200"]
assert membership_response["content"]["application/json"]["schema"]["$ref"].endswith("/MembershipUpdate")
print(f"PASS: {operations} operations; local refs, path parameters, unique operation IDs, error envelopes, credential/CSRF boundaries, Go Check fields, tenant roles and policy fixture enums agree.")
print("LIMIT: structural/cross-file check only; not full OpenAPI validation or runtime authorization acceptance.")
