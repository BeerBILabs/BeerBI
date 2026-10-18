import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { NetworkChart } from '../charts/NetworkChart';
import { resetUserCacheForTests, setCachedUser, userDataManager } from '@/lib/userCache';

const fetchMock = vi.fn();

beforeEach(() => {
  localStorage.clear();
  resetUserCacheForTests();
  userDataManager.clearInFlight();
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe('NetworkChart', () => {
  it('uses inline pair names without a /users request', async () => {
    render(
      <NetworkChart
        startDate="2026-01-01"
        endDate="2026-10-18"
        preloadedData={[{ giver: 'U1', recipient: 'U2', count: 3, giver_name: 'Alice Example', recipient_name: 'Bob Example' }]}
      />
    );
    expect(await screen.findByText('Alice Example')).toBeInTheDocument();
    expect(screen.getByText('Bob Example')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('resolves missing names through the batch endpoint', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ U1: { real_name: 'Alice Example', profile_image: null }, U2: { real_name: 'Bob Example', profile_image: null } }),
    });
    render(<NetworkChart startDate="2026-01-01" endDate="2026-10-18" preloadedData={[{ giver: 'U1', recipient: 'U2', count: 3 }]} />);

    expect(await screen.findByText('Alice Example')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(String(fetchMock.mock.calls[0][0])).toBe(`/api/proxy/users?ids=${encodeURIComponent('U1,U2')}`);
  });

  it('fetches pairs itself without preloaded data', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => [{ giver: 'U1', recipient: 'U2', count: 2, giver_name: 'Alice Example', recipient_name: 'Bob Example' }],
    });
    render(<NetworkChart startDate="2026-01-01" endDate="2026-10-18" limit={5} />);

    expect(await screen.findByText('Alice Example')).toBeInTheDocument();
    expect(String(fetchMock.mock.calls[0][0])).toBe('/api/proxy/stats/pairs?start=2026-01-01&end=2026-10-18&limit=5');
  });

  it('labels new rows at once from inline or cached names when the range changes', async () => {
    setCachedUser('U4', 'Dave Example', null);
    const { rerender } = render(
      <NetworkChart
        startDate="2026-01-01"
        endDate="2026-10-18"
        preloadedData={[{ giver: 'U1', recipient: 'U2', count: 3, giver_name: 'Alice Example', recipient_name: 'Bob Example' }]}
      />
    );
    expect(await screen.findByText('Alice Example')).toBeInTheDocument();

    rerender(
      <NetworkChart
        startDate="2025-01-01"
        endDate="2025-12-31"
        preloadedData={[{ giver: 'U3', recipient: 'U4', count: 1, giver_name: 'Carol Example' }]}
      />
    );
    // no raw IDs while resolveUsers runs
    expect(screen.getByText('Carol Example')).toBeInTheDocument();
    expect(screen.getByText('Dave Example')).toBeInTheDocument();
    expect(screen.queryByText('U3')).toBeNull();
  });

  it('shows the empty state', async () => {
    render(<NetworkChart startDate="2026-01-01" endDate="2026-10-18" preloadedData={[]} />);
    expect(await screen.findByText('No pair data available for this period')).toBeInTheDocument();
  });
});
