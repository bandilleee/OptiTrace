import type { components } from './api-schema';
export type Policy = components['schemas']['Policy'];
export type Check = components['schemas']['Check'];
export type Evaluation = components['schemas']['Evaluation'];
export type Identity = components['schemas']['Session'];
export type Benchmark = components['schemas']['Benchmark'];
export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function api<T>(
  path: string,
  options: {
    method?: string;
    body?: unknown;
    csrf?: string;
    idempotency?: string;
  } = {},
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.csrf) headers['X-CSRF-Token'] = options.csrf;
  if (options.idempotency) headers['Idempotency-Key'] = options.idempotency;
  const response = await fetch('/api/v1' + path, {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    credentials: 'same-origin',
    cache: 'no-store',
  });
  const value = response.status === 204 ? undefined : await response.json().catch(() => undefined);
  if (!response.ok)
    throw new APIError(
      response.status,
      value?.error?.message ?? `Request failed (${response.status})`,
    );
  return value as T;
}
export function label(value: string) {
  return value.replaceAll('_', ' ').replace(/^\w/, (s) => s.toUpperCase());
}
export function cost(value: number | null) {
  if (value === null) return 'Unconfirmed';
  return new Intl.NumberFormat('en-ZA', {
    style: 'currency',
    currency: 'ZAR',
    minimumFractionDigits: 4,
    maximumFractionDigits: 6,
  }).format(value / 1000000);
}
export function can(roles: string[], role: string) {
  return roles.includes(role);
}
export const scenarios = [
  'clear',
  'number_mismatch',
  'sim_change',
  'device_change',
  'all_adverse',
  'stale',
  'timeout',
  'rate_limit',
  'permission_denied',
  'source_failure',
  'correlated',
];
export const strategies = ['optitrace', 'full_sequential', 'full_parallel', 'staged', 'opa'];

export function duration(value: number | null) {
  return value === null ? 'Unconfirmed' : `${value.toFixed(1)} ms`;
}
