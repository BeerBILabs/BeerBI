import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, render, waitFor } from '@testing-library/react';
import { LeaderboardChart } from '../charts/LeaderboardChart';
import { resetUserCacheForTests, userDataManager } from '@/lib/userCache';

const fetchMock = vi.fn();

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

beforeEach(() => {
  localStorage.clear();
  resetUserCacheForTests();
  userDataManager.clearInFlight();
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  vi.stubGlobal('ResizeObserver', ResizeObserverStub);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe('LeaderboardChart', () => {
  it('makes no request when the rows carry names', async () => {
    render(
      <LeaderboardChart
        startDate="2026-01-01"
        endDate="2026-10-18"
        type="givers"
        preloadedData={[{ userId: 'U1', count: 3, name: 'Alice Example' }]}
      />
    );
    await act(async () => {});
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('resolves only the shown top 10 IDs when names are missing', async () => {
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({}) });
    const rows = Array.from({ length: 12 }, (_, i) => ({ userId: `U${i}`, count: 20 - i }));
    render(<LeaderboardChart startDate="2026-01-01" endDate="2026-10-18" type="recipients" preloadedData={rows} />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const ids = new URL(String(fetchMock.mock.calls[0][0]), 'http://localhost').searchParams.get('ids');
    expect(ids?.split(',')).toHaveLength(10);
  });

  it('fetches /stats/top without preloaded data', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ givers: [], recipients: [{ userId: 'U1', count: 1, name: 'Alice Example' }] }),
    });
    render(<LeaderboardChart startDate="2026-01-01" endDate="2026-10-18" type="recipients" limit={5} />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(String(fetchMock.mock.calls[0][0])).toBe('/api/proxy/stats/top?start=2026-01-01&end=2026-10-18&limit=5');
  });
});
