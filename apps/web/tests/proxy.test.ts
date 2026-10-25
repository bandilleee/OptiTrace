import { afterEach, describe, it, expect, vi } from 'vitest';
import { proxy } from '../src/lib/proxy';
afterEach(() => vi.unstubAllGlobals());
describe('same-origin API proxy', () => {
  it('returns a retryable API error for host wake pages without replaying a mutation', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(
        new Response('<html>Starting</html>', { headers: { 'content-type': 'text/html' } }),
      );
    vi.stubGlobal('fetch', fetcher);
    const result = await proxy(
      new Request('http://localhost:3000/api/v1/evaluations', { method: 'POST', body: '{}' }),
      ['v1', 'evaluations'],
    );
    expect(result.status).toBe(503);
    expect(result.headers.get('retry-after')).toBe('30');
    expect((await result.json()).error.code).toBe('upstream_unavailable');
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
  it('handles cold-start transport failure without exposing local setup instructions', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('private upstream detail')));
    const result = await proxy(new Request('http://localhost:3000/api/v1/me'), ['v1', 'me']);
    expect(result.status).toBe(503);
    expect(await result.text()).not.toMatch(/private upstream|local API/);
  });
  it('limits streamed bodies before forwarding upstream', async () => {
    const fetcher = vi.fn();
    vi.stubGlobal('fetch', fetcher);
    const result = await proxy(
      new Request('http://localhost:3000/api/v1/test', {
        method: 'POST',
        body: 'a'.repeat(1_048_577),
      }),
      ['v1', 'test'],
    );
    expect(result.status).toBe(413);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it('rejects non-API and path traversal destinations before fetch', async () => {
    const fetcher = vi.fn();
    vi.stubGlobal('fetch', fetcher);
    for (const path of [
      ['https:', 'evil.example'],
      ['v1', '..', 'secrets'],
      ['v1', 'a/b'],
    ])
      expect((await proxy(new Request('http://localhost:3000/api/v1'), path)).status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it('forwards session/CSRF and idempotency headers while dropping caller-controlled host headers', async () => {
    const response = new Response('{"ok":true}', {
      headers: {
        'content-type': 'application/json',
        'set-cookie': 'optitrace_session=abc; HttpOnly; Path=/',
        'x-secret-upstream': 'hidden',
      },
    });
    const fetcher = vi.fn().mockResolvedValue(response);
    vi.stubGlobal('fetch', fetcher);
    const result = await proxy(
      new Request('http://localhost:3000/api/v1/tenants/t/evaluations?test=1', {
        method: 'POST',
        headers: {
          cookie: 'optitrace_session=abc',
          origin: 'http://localhost:3000',
          'x-csrf-token': 'csrf',
          'x-forwarded-host': 'evil.example',
          'idempotency-key': 'k',
        },
        body: '{}',
      }),
      ['v1', 'tenants', 't', 'evaluations'],
    );
    const [url, options] = fetcher.mock.calls[0];
    expect(String(url)).toBe('http://localhost:8080/v1/tenants/t/evaluations?test=1');
    expect(options.redirect).toBe('manual');
    expect(options.headers.get('cookie')).toContain('optitrace_session');
    expect(options.headers.get('x-csrf-token')).toBe('csrf');
    expect(options.headers.has('x-forwarded-host')).toBe(false);
    expect(result.headers.get('set-cookie')).toContain('HttpOnly');
    expect(result.headers.has('x-secret-upstream')).toBe(false);
  });
  it('preserves provider redirects without following them on the server', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(null, {
          status: 302,
          headers: { location: 'http://localhost:8090/authorize' },
        }),
      ),
    );
    const result = await proxy(new Request('http://localhost:3000/api/v1/auth/login'), [
      'v1',
      'auth',
      'login',
    ]);
    expect(result.status).toBe(302);
    expect(result.headers.get('location')).toBe('http://localhost:8090/authorize');
  });
});
