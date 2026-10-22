import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const spec = JSON.parse(readFileSync(new URL('../contracts/openapi.json', import.meta.url), 'utf8'));
const lines = [
  '// Generated from contracts/openapi.json. Do not edit.',
  "import type { paths } from './api-schema';",
  "import { api } from './api';",
  '',
];
for (const [route, methods] of Object.entries(spec.paths)) {
  if (!route.startsWith('/v1/')) continue;
  for (const [method, operation] of Object.entries(methods)) {
    const name = operation.operationId;
    if (!name) continue;
    const opType = `paths[${JSON.stringify(route)}][${JSON.stringify(method)}]`;
    const parameters = (operation.parameters ?? []).filter(p => p.in === 'path');
    const fields = [];
    if (parameters.length) fields.push(`path: ${opType}['parameters']['path']`);
    if (operation.requestBody) fields.push(`body: ${opType}['requestBody']['content']['application/json']`);
    const idempotency = (operation.parameters ?? []).some(p => p.name === 'Idempotency-Key' && p.required);
    fields.push(`csrf?: string`, `idempotency${idempotency ? '' : '?'}: string`);
    const required = parameters.length || operation.requestBody || idempotency;
    const input = `input: { ${fields.join('; ')} }${required ? '' : ' = {}'}`;
    let endpoint = JSON.stringify(route.slice(3));
    for (const p of parameters) endpoint += `.replace(${JSON.stringify(`{${p.name}}`)}, encodeURIComponent(String(input.path.${p.name})))`;
    const success = Object.keys(operation.responses).find(code => /^2\d\d$/.test(code));
    if (!success) {
      lines.push(`export function ${name}Url(${input}): string { return '/api/v1' + ${endpoint}; }`, '');
      continue;
    }
    const jsonResponse = operation.responses[success]?.content?.['application/json'];
    const response = jsonResponse ? `${opType}['responses'][${success}]['content']['application/json']` : 'void';
    const body = operation.requestBody ? ', body: input.body' : '';
    lines.push(`export function ${name}(${input}): Promise<${response}> {`,
      `  return api<${response}>(${endpoint}, { method: ${JSON.stringify(method.toUpperCase())}, csrf: input.csrf, idempotency: input.idempotency${body} });`, '}', '');
  }
}
const output = new URL('../apps/web/src/lib/sdk.ts', import.meta.url);
writeFileSync(output, lines.join('\n'));
console.log(`Generated typed SDK: ${fileURLToPath(output)}`);
