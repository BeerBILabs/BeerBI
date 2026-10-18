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
