import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import { Leaderboard, leaderboardUrl } from '../Leaderboard';
import { resetUserCacheForTests, userDataManager } from '@/lib/userCache';

const fetchMock = vi.fn();

function jsonResponse(body: unknown, ok = true) {
  return { ok, statusText: ok ? 'OK' : 'Bad Gateway', json: async () => body };
}

const EMPTY = { current: { givers: [], recipients: [] }, previous: null };

function urls(): string[] {
  return fetchMock.mock.calls.map(([url]) => String(url));
}

beforeEach(() => {
  localStorage.clear();
  resetUserCacheForTests();
  userDataManager.clearInFlight();
  fetchMock.mockReset();
  fetchMock.mockResolvedValue(jsonResponse(EMPTY));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('Leaderboard all-time date range', () => {
  // Regression for audit C7: toISOString() made "today" yesterday between 00:00 and 02:00 Berlin time.
  it('uses the local calendar day as end date shortly after local midnight', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    // 00:30 in Berlin on 2026-03-15 == 23:30 UTC on 2026-03-14
    vi.setSystemTime(new Date('2026-03-15T00:30:00+01:00'));

    render(<Leaderboard year={null} quarter={null} />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const url = urls()[0];
    expect(url).toContain('start=2020-01-01');
    expect(url).toContain('end=2026-03-15');
    expect(url).not.toContain('prev_start');
  });
});

describe('Leaderboard quarter date range', () => {
  it('requests the local quarter boundaries', async () => {
    render(<Leaderboard year={2026} quarter={1} showRankChange={false} />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const url = urls()[0];
    expect(url).toContain('start=2026-01-01');
    expect(url).toContain('end=2026-03-31');
    expect(url).not.toContain('prev_start');
  });

  it('asks for the previous quarter in the same request when showing rank changes (F2)', () => {
    const url = new URL(leaderboardUrl({ year: 2026, quarter: 1, showRankChange: true }), 'http://localhost');
    expect(url.pathname).toBe('/api/leaderboard');
    expect(Object.fromEntries(url.searchParams)).toEqual({
      limit: '100',
      start: '2026-01-01',
      end: '2026-03-31',
      prev_start: '2025-10-01',
      prev_end: '2025-12-31',
    });
  });
});

describe('Leaderboard data', () => {
  it('renders inline names from the stats response without a /users request', async () => {
    fetchMock.mockResolvedValue(jsonResponse({
      current: {
        givers: [{ userId: 'U1', count: 5, name: 'Alice Example', avatar: null }],
        recipients: [{ userId: 'U2', count: 4, name: 'Bob Example', avatar: null }],
      },
      previous: { givers: [{ userId: 'U1', count: 1 }], recipients: [] },
    }));

    render(<Leaderboard year={2026} quarter={2} showRankChange />);

    expect(await screen.findByText('Alice Example')).toBeInTheDocument();
    expect(screen.getByText('Bob Example')).toBeInTheDocument();
    expect(screen.getByLabelText('No change')).toBeInTheDocument(); // U1 was #1 before as well
    expect(screen.getByLabelText('New this quarter')).toBeInTheDocument(); // U2 is new
    expect(urls()).toHaveLength(1);
    expect(urls()[0]).toMatch(/^\/api\/leaderboard\?/);
  });

  it('falls back to one batch lookup when the response has no names', async () => {
    fetchMock.mockImplementation(async (url: string) => {
      if (url.startsWith('/api/leaderboard')) {
        return jsonResponse({
          current: { givers: [{ userId: 'U1', count: 5 }], recipients: [{ userId: 'U2', count: 4 }, { userId: 'U1', count: 1 }] },
          previous: null,
        });
      }
      return jsonResponse({ U1: { real_name: 'Alice Example', profile_image: null } });
    });

    render(<Leaderboard dateRange={{ start: '2026-01-01', end: '2026-10-18' }} showRankChange={false} />);

    expect((await screen.findAllByText('Alice Example')).length).toBeGreaterThan(0);
    expect(screen.getByText('U2')).toBeInTheDocument(); // unknown user: ID as fallback
    const userCalls = urls().filter((u) => u.startsWith('/api/proxy/users'));
    expect(userCalls).toEqual([`/api/proxy/users?ids=${encodeURIComponent('U1,U2')}`]);
  });

  it('shows an error when the request fails', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, false));
    render(<Leaderboard year={null} quarter={null} />);
    expect(await screen.findByText('Failed to fetch top users')).toBeInTheDocument();
  });

  it('shows the empty state', async () => {
    render(<Leaderboard year={2026} quarter={1} />);
    expect(await screen.findByText(/No beers recorded for this quarter/)).toBeInTheDocument();
  });

  it('keeps the previous ranking visible while a new range loads', async () => {
    let release: (v: unknown) => void = () => {};
    fetchMock.mockResolvedValueOnce(jsonResponse({
      current: { givers: [{ userId: 'U1', count: 5, name: 'Alice Example' }], recipients: [] },
      previous: null,
    }));
    fetchMock.mockImplementationOnce(() => new Promise((resolve) => { release = resolve; }));

    const { rerender, container } = render(<Leaderboard dateRange={{ start: '2026-01-01', end: '2026-10-18' }} />);
    expect(await screen.findByText('Alice Example')).toBeInTheDocument();

    rerender(<Leaderboard dateRange={{ start: '2026-10-01', end: '2026-10-18' }} />);
    expect(screen.getByText('Alice Example')).toBeInTheDocument();
    expect(container.querySelector('[aria-busy="true"]')).not.toBeNull();

    await act(async () => release(jsonResponse({
      current: { givers: [{ userId: 'U3', count: 1, name: 'Carol Example' }], recipients: [] },
      previous: null,
    })));
    expect(await screen.findByText('Carol Example')).toBeInTheDocument();
    expect(screen.queryByText('Alice Example')).toBeNull();
  });

  it('does not refetch when the same range is passed as a new object', async () => {
    const { rerender } = render(<Leaderboard dateRange={{ start: '2026-01-01', end: '2026-10-18' }} />);
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    rerender(<Leaderboard dateRange={{ start: '2026-01-01', end: '2026-10-18' }} />);
    await screen.findByText(/No beers recorded/);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
