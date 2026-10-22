import { afterEach, describe, it, expect, vi } from 'vitest';
import { api, cost, duration } from '../src/lib/api';
afterEach(() => vi.unstubAllGlobals());
describe('API boundary', () => {
  it('sends the CSRF and idempotency headers and handles structured outcomes', async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ outcome: 'indeterminate' }));
    vi.stubGlobal('fetch', fetcher);
    await expect(
      api('/tenants/t/evaluations', {
        method: 'POST',
        body: { scenario: 'stale' },
        csrf: 'token',
        idempotency: 'request',
      }),
    ).resolves.toEqual({ outcome: 'indeterminate' });
    expect(fetcher).toHaveBeenCalledWith(
      '/api/v1/tenants/t/evaluations',
      expect.objectContaining({
        credentials: 'same-origin',
        headers: expect.objectContaining({ 'X-CSRF-Token': 'token', 'Idempotency-Key': 'request' }),
        body: '{"scenario":"stale"}',
      }),
    );
  });
  it('surfaces server authorization failures rather than returning success', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          Response.json(
            { error: { message: 'Reviewer cannot approve their own version' } },
            { status: 403 },
          ),
        ),
    );
    await expect(api('/policies/p/approve', { method: 'POST' })).rejects.toMatchObject({
      status: 403,
      message: 'Reviewer cannot approve their own version',
    });
  });
  it('handles revoked credentials returning no content', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })));
    await expect(api('/credentials/k', { method: 'DELETE' })).resolves.toBeUndefined();
  });
  it('converts microrand to rand using the host ZA decimal convention', () => {
    expect(cost(1000000)).toMatch(/(?:R|ZAR)\s*1[.,]0000/);
  });
  it('never presents unconfirmed execution cost or latency as zero', () => {
    expect(cost(null)).toBe('Unconfirmed');
    expect(duration(null)).toBe('Unconfirmed');
  });
});
