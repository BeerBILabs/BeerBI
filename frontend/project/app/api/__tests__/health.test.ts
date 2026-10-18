// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { GET } from '../health/route';

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  vi.stubEnv('BACKEND_BASE', 'http://backend.test:8080');
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('GET /api/health', () => {
  it('checks BACKEND_BASE/healthz', async () => {
    fetchMock.mockResolvedValue(new Response('ok'));
    const resp = await GET();
    expect(fetchMock.mock.calls[0][0]).toBe('http://backend.test:8080/healthz');
    expect(resp.status).toBe(200);
    expect(await resp.json()).toEqual({ status: 'ok', backend: 'http://backend.test:8080' });
  });

  it('reports an unhealthy or unreachable backend as 502', async () => {
    fetchMock.mockResolvedValueOnce(new Response('down', { status: 503 }));
    expect((await GET()).status).toBe(502);

    fetchMock.mockRejectedValueOnce(new Error('ECONNREFUSED'));
    const resp = await GET();
    expect(resp.status).toBe(502);
    expect((await resp.json()).status).toBe('unreachable');
  });
});
