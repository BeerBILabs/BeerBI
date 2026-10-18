// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';
import { GET } from '../leaderboard/route';

const fetchMock = vi.fn();

function get(query: string) {
  return GET(new NextRequest(`http://localhost:3000/api/leaderboard?${query}`));
}

function top(id: string) {
  return { givers: [{ userId: id, count: 2 }], recipients: [{ userId: `${id}R`, count: 1 }] };
}

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  vi.stubEnv('BACKEND_BASE', 'http://backend.test:8080');
  vi.stubEnv('API_TOKEN', 'test-token');
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('GET /api/leaderboard', () => {
  it('returns the current period only when no comparison is asked for', async () => {
    fetchMock.mockResolvedValue(Response.json(top('U1')));

    const resp = await get('start=2026-01-01&end=2026-10-18');

    expect(resp.status).toBe(200);
    expect(await resp.json()).toEqual({ current: top('U1'), previous: null });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('http://backend.test:8080/api/stats/top?start=2026-01-01&end=2026-10-18&limit=100');
    expect((init.headers as Headers).get('authorization')).toBe('Bearer test-token');
  });

  it('fetches current and previous period in parallel', async () => {
    const pending: Array<(r: Response) => void> = [];
    fetchMock.mockImplementation(() => new Promise<Response>((resolve) => pending.push(resolve)));

    const respPromise = get('start=2026-10-01&end=2026-12-31&prev_start=2026-07-01&prev_end=2026-09-30&limit=50');
    await vi.waitFor(() => expect(pending).toHaveLength(2)); // both started before either finished
    pending[1](Response.json(top('P')));
    pending[0](Response.json(top('C')));

    const body = await (await respPromise).json();
    expect(body).toEqual({ current: top('C'), previous: top('P') });
    expect(String(fetchMock.mock.calls[1][0])).toContain('start=2026-07-01&end=2026-09-30&limit=50');
  });

  it('shows the ranking without comparison when the previous period fails', async () => {
    fetchMock
      .mockResolvedValueOnce(Response.json(top('C')))
      .mockResolvedValueOnce(new Response('boom', { status: 500 }));

    const body = await (await get('start=2026-10-01&end=2026-12-31&prev_start=2026-07-01&prev_end=2026-09-30')).json();
    expect(body).toEqual({ current: top('C'), previous: null });
  });

  it('returns 502 when the current period fails', async () => {
    fetchMock.mockResolvedValue(new Response('boom', { status: 500 }));
    const resp = await get('start=2026-10-01&end=2026-12-31');
    expect(resp.status).toBe(502);
    expect((await resp.json()).error).toContain('500');
  });

  it('treats missing lists as empty', async () => {
    fetchMock.mockResolvedValue(Response.json({ givers: null }));
    const body = await (await get('start=2026-10-01&end=2026-12-31')).json();
    expect(body.current).toEqual({ givers: [], recipients: [] });
  });

  it.each([
    ['start=2026-1-1&end=2026-03-31'],
    ['end=2026-03-31'],
    ['start=2026-01-01&end=2026-03-31&prev_start=2025-10-01'],
    ['start=2026-01-01&end=2026-03-31&prev_start=x&prev_end=2025-12-31'],
    ['start=2026-01-01&end=2026-03-31&limit=0'],
    ['start=2026-01-01&end=2026-03-31&limit=101'],
    ['start=2026-01-01&end=2026-03-31&limit=abc'],
  ])('rejects invalid query %s', async (query) => {
    const resp = await get(query);
    expect(resp.status).toBe(400);
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

describe('GET /api/leaderboard caching', () => {
  function cached(body: unknown, cacheControl: string | null) {
    return Response.json(body, cacheControl ? { headers: { 'Cache-Control': cacheControl } } : undefined);
  }
  const both = 'start=2026-10-01&end=2026-12-31&prev_start=2026-07-01&prev_end=2026-09-30';

  it('uses the shorter backend max-age of both periods, always private', async () => {
    fetchMock
      .mockResolvedValueOnce(cached(top('C'), 'private, max-age=30'))
      .mockResolvedValueOnce(cached(top('P'), 'private, max-age=3600'));
    const resp = await get(both);
    expect(resp.headers.get('cache-control')).toBe('private, max-age=30');
  });

  it('takes the max-age of the current period alone when no comparison is asked for', async () => {
    fetchMock.mockResolvedValue(cached(top('C'), 'public, max-age=3600'));
    const resp = await get('start=2025-01-01&end=2025-03-31');
    expect(resp.headers.get('cache-control')).toBe('private, max-age=3600');
  });

  it.each([
    ['no header on one part', 'private, max-age=3600', null],
    ['no-store on one part', 'private, max-age=3600', 'no-store'],
    ['no-cache with max-age', 'no-cache, max-age=60', 'private, max-age=60'],
    ['max-age=0', 'max-age=0', 'max-age=60'],
  ])('is not cached with %s', async (_label, a, b) => {
    fetchMock.mockResolvedValueOnce(cached(top('C'), a)).mockResolvedValueOnce(cached(top('P'), b));
    const resp = await get(both);
    expect(resp.headers.get('cache-control')).toBe('private, no-store');
  });

  it('is not cached when the comparison period failed, so the arrows come back on reload', async () => {
    fetchMock
      .mockResolvedValueOnce(cached(top('C'), 'private, max-age=3600'))
      .mockResolvedValueOnce(new Response('boom', { status: 500 }));
    const resp = await get(both);
    expect(resp.headers.get('cache-control')).toBe('private, no-store');
  });

  function tagged(body: unknown, etag: string, cacheControl = 'private, max-age=60') {
    return Response.json(body, { headers: { 'Cache-Control': cacheControl, ETag: etag } });
  }
  function getWith(query: string, ifNoneMatch: string) {
    return GET(
      new NextRequest(`http://localhost:3000/api/leaderboard?${query}`, { headers: { 'If-None-Match': ifNoneMatch } })
    );
  }

  it('derives an ETag from the backend ETags and answers a match with 304', async () => {
    fetchMock.mockResolvedValueOnce(tagged(top('C'), '"c1"')).mockResolvedValueOnce(tagged(top('P'), '"p1"'));
    const first = await get(both);
    const etag = first.headers.get('etag');
    expect(etag).toMatch(/^"lb-[0-9a-f]{32}"$/);

    fetchMock.mockResolvedValueOnce(tagged(top('C'), '"c1"')).mockResolvedValueOnce(tagged(top('P'), '"p1"'));
    const second = await getWith(both, `W/${etag}`);
    expect(second.status).toBe(304);
    expect(second.headers.get('etag')).toBe(etag);
    expect(second.headers.get('cache-control')).toBe('private, max-age=60');
    expect(await second.text()).toBe('');
  });

  it('sends the full body when a backend ETag changed', async () => {
    fetchMock.mockResolvedValueOnce(tagged(top('C'), '"c1"')).mockResolvedValueOnce(tagged(top('P'), '"p1"'));
    const etag = (await get(both)).headers.get('etag')!;

    fetchMock.mockResolvedValueOnce(tagged(top('C'), '"c2"')).mockResolvedValueOnce(tagged(top('P'), '"p1"'));
    const resp = await getWith(both, etag);
    expect(resp.status).toBe(200);
    expect(resp.headers.get('etag')).not.toBe(etag);
    expect((await resp.json()).current.givers[0].userId).toBe('C');
  });

  it('sends no ETag when a part has none or the response is not cacheable', async () => {
    fetchMock.mockResolvedValueOnce(tagged(top('C'), '"c1"')).mockResolvedValueOnce(cached(top('P'), 'private, max-age=60'));
    expect((await get(both)).headers.get('etag')).toBeNull();

    fetchMock.mockResolvedValueOnce(tagged(top('C'), '"c1"', 'no-store')).mockResolvedValueOnce(tagged(top('P'), '"p1"'));
    const resp = await getWith(both, '*');
    expect(resp.status).toBe(200);
    expect(resp.headers.get('etag')).toBeNull();
  });
});
