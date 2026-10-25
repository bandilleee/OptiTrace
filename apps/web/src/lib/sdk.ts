// Generated from contracts/openapi.json. Do not edit.
import type { paths } from './api-schema';
import { api } from './api';

export function loginUrl(input: { csrf?: string; idempotency?: string } = {}): string { return '/api/v1' + "/auth/login"; }

export function callbackUrl(input: { csrf?: string; idempotency?: string } = {}): string { return '/api/v1' + "/auth/callback"; }

export function getSession(input: { csrf?: string; idempotency?: string } = {}): Promise<paths["/v1/auth/me"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/auth/me"]["get"]['responses'][200]['content']['application/json']>("/auth/me", { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function logout(input: { csrf?: string; idempotency?: string } = {}): Promise<void> {
  return api<void>("/auth/logout", { method: "POST", csrf: input.csrf, idempotency: input.idempotency });
}

export function listPolicies(input: { path: paths["/v1/tenants/{tenant}/policies"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/policies"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/policies"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/policies".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function createPolicy(input: { path: paths["/v1/tenants/{tenant}/policies"]["post"]['parameters']['path']; body: paths["/v1/tenants/{tenant}/policies"]["post"]['requestBody']['content']['application/json']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/policies"]["post"]['responses'][201]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/policies"]["post"]['responses'][201]['content']['application/json']>("/tenants/{tenant}/policies".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}

export function listEvaluations(input: { path: paths["/v1/tenants/{tenant}/evaluations"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/evaluations"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/evaluations"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/evaluations".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function createEvaluation(input: { path: paths["/v1/tenants/{tenant}/evaluations"]["post"]['parameters']['path']; body: paths["/v1/tenants/{tenant}/evaluations"]["post"]['requestBody']['content']['application/json']; csrf?: string; idempotency: string }): Promise<paths["/v1/tenants/{tenant}/evaluations"]["post"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/evaluations"]["post"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/evaluations".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}

export function listBenchmarks(input: { path: paths["/v1/tenants/{tenant}/benchmarks"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/benchmarks"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/benchmarks"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/benchmarks".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function createBenchmark(input: { path: paths["/v1/tenants/{tenant}/benchmarks"]["post"]['parameters']['path']; body: paths["/v1/tenants/{tenant}/benchmarks"]["post"]['requestBody']['content']['application/json']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/benchmarks"]["post"]['responses'][202]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/benchmarks"]["post"]['responses'][202]['content']['application/json']>("/tenants/{tenant}/benchmarks".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}

export function listMemberships(input: { path: paths["/v1/tenants/{tenant}/memberships"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/memberships"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/memberships"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/memberships".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function listInvitations(input: { path: paths["/v1/tenants/{tenant}/invitations"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/invitations"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/invitations"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/invitations".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function createInvitation(input: { path: paths["/v1/tenants/{tenant}/invitations"]["post"]['parameters']['path']; body: paths["/v1/tenants/{tenant}/invitations"]["post"]['requestBody']['content']['application/json']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/invitations"]["post"]['responses'][201]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/invitations"]["post"]['responses'][201]['content']['application/json']>("/tenants/{tenant}/invitations".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}

export function listCredentials(input: { path: paths["/v1/tenants/{tenant}/credentials"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/credentials"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/credentials"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/credentials".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function createCredential(input: { path: paths["/v1/tenants/{tenant}/credentials"]["post"]['parameters']['path']; body: paths["/v1/tenants/{tenant}/credentials"]["post"]['requestBody']['content']['application/json']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/credentials"]["post"]['responses'][201]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/credentials"]["post"]['responses'][201]['content']['application/json']>("/tenants/{tenant}/credentials".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}

export function listAudit(input: { path: paths["/v1/tenants/{tenant}/audit"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/audit"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/audit"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/audit".replace("{tenant}", encodeURIComponent(String(input.path.tenant))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function getEvaluation(input: { path: paths["/v1/tenants/{tenant}/evaluations/{id}"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/evaluations/{id}"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/evaluations/{id}"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/evaluations/{id}".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function getBenchmark(input: { path: paths["/v1/tenants/{tenant}/benchmarks/{id}"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/benchmarks/{id}"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/benchmarks/{id}"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/benchmarks/{id}".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function submitPolicy(input: { path: paths["/v1/tenants/{tenant}/policies/{id}/submit"]["post"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/policies/{id}/submit"]["post"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/policies/{id}/submit"]["post"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/policies/{id}/submit".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency });
}

export function approvePolicy(input: { path: paths["/v1/tenants/{tenant}/policies/{id}/approve"]["post"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/policies/{id}/approve"]["post"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/policies/{id}/approve"]["post"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/policies/{id}/approve".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency });
}

export function activatePolicy(input: { path: paths["/v1/tenants/{tenant}/policies/{id}/activate"]["post"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/policies/{id}/activate"]["post"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/policies/{id}/activate"]["post"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/policies/{id}/activate".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "POST", csrf: input.csrf, idempotency: input.idempotency });
}

export function updateMembership(input: { path: paths["/v1/tenants/{tenant}/memberships/{id}"]["patch"]['parameters']['path']; body: paths["/v1/tenants/{tenant}/memberships/{id}"]["patch"]['requestBody']['content']['application/json']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/memberships/{id}"]["patch"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/memberships/{id}"]["patch"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/memberships/{id}".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "PATCH", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}

export function revokeCredential(input: { path: paths["/v1/tenants/{tenant}/credentials/{id}"]["delete"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<void> {
  return api<void>("/tenants/{tenant}/credentials/{id}".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "DELETE", csrf: input.csrf, idempotency: input.idempotency });
}

export function exportBenchmark(input: { path: paths["/v1/tenants/{tenant}/benchmarks/{id}/export"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/benchmarks/{id}/export"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/benchmarks/{id}/export"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/benchmarks/{id}/export".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function getPolicy(input: { path: paths["/v1/tenants/{tenant}/policies/{id}"]["get"]['parameters']['path']; csrf?: string; idempotency?: string }): Promise<paths["/v1/tenants/{tenant}/policies/{id}"]["get"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/tenants/{tenant}/policies/{id}"]["get"]['responses'][200]['content']['application/json']>("/tenants/{tenant}/policies/{id}".replace("{tenant}", encodeURIComponent(String(input.path.tenant))).replace("{id}", encodeURIComponent(String(input.path.id))), { method: "GET", csrf: input.csrf, idempotency: input.idempotency });
}

export function runPublicDemo(input: { body: paths["/v1/demo/evaluations"]["post"]['requestBody']['content']['application/json']; csrf?: string; idempotency?: string }): Promise<paths["/v1/demo/evaluations"]["post"]['responses'][200]['content']['application/json']> {
  return api<paths["/v1/demo/evaluations"]["post"]['responses'][200]['content']['application/json']>("/demo/evaluations", { method: "POST", csrf: input.csrf, idempotency: input.idempotency, body: input.body });
}
