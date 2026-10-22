const REQUEST_HEADERS = [
  'accept',
  'content-type',
  'cookie',
  'origin',
  'x-csrf-token',
  'idempotency-key',
  'authorization',
];
const MAX_BODY = 1_048_576;
async function limitedBody(request: Request) {
  if (Number(request.headers.get('content-length') ?? 0) > MAX_BODY)
    throw new RangeError('payload_too_large');
  if (!request.body) return undefined;
  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    length += value.byteLength;
    if (length > MAX_BODY) {
      await reader.cancel();
      throw new RangeError('payload_too_large');
    }
    chunks.push(value);
  }
  const body = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    body.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return body.buffer;
}
export async function proxy(request: Request, parts: string[]) {
  if (parts[0] !== 'v1' || parts.some((p) => p === '.' || p === '..' || /[\\/?#]/.test(p)))
    return Response.json(
      { error: { code: 'invalid_path', message: 'Unknown API path' } },
      { status: 400 },
    );
  const base = new URL(process.env.API_BASE_URL ?? 'http://localhost:8080');
  if (!['http:', 'https:'].includes(base.protocol))
    return Response.json({ error: { message: 'Invalid upstream configuration' } }, { status: 503 });
  const target = new URL('/' + parts.map(encodeURIComponent).join('/'), base);
  target.search = new URL(request.url).search;
  const headers = new Headers();
  for (const key of REQUEST_HEADERS) {
    const value = request.headers.get(key);
    if (value) headers.set(key, value);
  }
  try {
    const result = await fetch(target, {
      method: request.method,
      headers,
      body: ['GET', 'HEAD'].includes(request.method) ? undefined : await limitedBody(request),
      redirect: 'manual',
      cache: 'no-store',
      signal: AbortSignal.timeout(120000),
    });
    const responseHeaders = new Headers({ 'Cache-Control': 'no-store' });
    for (const key of ['content-type', 'location', 'content-disposition', 'retry-after']) {
      const value = result.headers.get(key);
      if (value) responseHeaders.set(key, value);
    }
    for (const value of result.headers.getSetCookie()) responseHeaders.append('set-cookie', value);
    return new Response(result.body, { status: result.status, headers: responseHeaders });
  } catch (error) {
    if (error instanceof RangeError && error.message === 'payload_too_large')
      return Response.json(
        { error: { code: 'payload_too_large', message: 'API request exceeds the 1 MiB limit.' } },
        { status: 413 },
      );
    return Response.json(
      {
        error: {
          code: 'upstream_unavailable',
          message: 'The verification service is unavailable. Check the local API and try again.',
        },
      },
      { status: 503 },
    );
  }
}
