// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';
import { GET, HEAD } from '../proxy/[...path]/route';

const fetchMock = vi.fn();

function call(handler: typeof GET, url: string, init?: ConstructorParameters<typeof NextRequest>[1], path = ['stats', 'top']) {
  return handler(new NextRequest(url, init), { params: Promise.resolve({ path }) });
}

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  vi.stubEnv('BACKEND_BASE', 'http://backend.test:8080/');
  vi.stubEnv('API_TOKEN', 'test-token');
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('proxy route', () => {
  it('calls BACKEND_BASE with the path and query and the API_TOKEN', async () => {
    fetchMock.mockResolvedValue(new Response('{"ok":true}', { headers: { 'Content-Type': 'application/json' } }));

    const resp = await call(GET, 'http://localhost:3000/api/proxy/stats/top?start=2026-01-01&end=2026-03-31');

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('http://backend.test:8080/api/stats/top?start=2026-01-01&end=2026-03-31');
    expect((init.headers as Headers).get('authorization')).toBe('Bearer test-token');
    expect(resp.status).toBe(200);
    expect(await resp.json()).toEqual({ ok: true });
  });

  it('never forwards a client Authorization header', async () => {
    fetchMock.mockResolvedValue(new Response('{}'));
    await call(GET, 'http://localhost:3000/api/proxy/stats/top', { headers: { Authorization: 'Bearer from-browser' } });

    const headers = fetchMock.mock.calls[0][1].headers as Headers;
    expect(headers.get('authorization')).toBe('Bearer test-token');

    vi.stubEnv('API_TOKEN', '');
    await call(GET, 'http://localhost:3000/api/proxy/stats/top', { headers: { Authorization: 'Bearer from-browser' } });
    expect((fetchMock.mock.calls[1][1].headers as Headers).has('authorization')).toBe(false);
  });

  it('forwards caching headers both ways and passes 304 without a body', async () => {
    fetchMock.mockResolvedValue(new Response(null, {
      status: 304,
      headers: { ETag: '"abc"', 'Cache-Control': 'max-age=60', 'Content-Encoding': 'gzip', 'Set-Cookie': 'x=1' },
    }));

    const resp = await call(GET, 'http://localhost:3000/api/proxy/stats/top', { headers: { 'If-None-Match': '"abc"' } });

    expect((fetchMock.mock.calls[0][1].headers as Headers).get('if-none-match')).toBe('"abc"');
    expect(resp.status).toBe(304);
    expect(resp.headers.get('etag')).toBe('"abc"');
    expect(resp.headers.get('cache-control')).toBe('max-age=60');
    expect(resp.headers.get('content-encoding')).toBeNull();
    expect(resp.headers.get('set-cookie')).toBeNull();
    expect(resp.body).toBeNull();
  });

  it('streams the backend body through', async () => {
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('[1,'));
        controller.enqueue(new TextEncoder().encode('2]'));
        controller.close();
      },
    });
    fetchMock.mockResolvedValue(new Response(stream, { status: 200, headers: { 'Content-Type': 'application/json' } }));

    const resp = await call(GET, 'http://localhost:3000/api/proxy/stats/heatmap', undefined, ['stats', 'heatmap']);

    expect(resp.headers.get('content-type')).toBe('application/json');
    expect(await resp.text()).toBe('[1,2]');
  });

  it('keeps the backend status and defaults the content type', async () => {
    fetchMock.mockResolvedValue(new Response('nope', { status: 401, headers: { 'Content-Type': '' } }));
    const resp = await call(GET, 'http://localhost:3000/api/proxy/user?user=U1', undefined, ['user']);
    expect(resp.status).toBe(401);
    expect(resp.headers.get('content-type')).toBe('text/plain');
  });

  it('answers HEAD without a body', async () => {
    fetchMock.mockResolvedValue(new Response('{}', { headers: { 'Content-Type': 'application/json' } }));
    const resp = await call(HEAD, 'http://localhost:3000/api/proxy/stats/top', { method: 'HEAD' });
    expect(resp.status).toBe(200);
    expect(resp.body).toBeNull();
  });

  it('returns 502 when the backend is unreachable', async () => {
    fetchMock.mockRejectedValue(new Error('connect ECONNREFUSED'));
    const resp = await call(GET, 'http://localhost:3000/api/proxy/stats/top');
    expect(resp.status).toBe(502);
    expect(await resp.json()).toEqual({ error: 'connect ECONNREFUSED' });
  });

  it('falls back to localhost when BACKEND_BASE is unset', async () => {
    vi.stubEnv('BACKEND_BASE', '');
    fetchMock.mockResolvedValue(new Response('{}'));
    await call(GET, 'http://localhost:3000/api/proxy/health', undefined, ['health']);
    expect(fetchMock.mock.calls[0][0]).toBe('http://localhost:8080/api/health');
  });
});
