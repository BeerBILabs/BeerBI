import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, waitFor } from '@testing-library/react';
import { Leaderboard } from '../Leaderboard';

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  fetchMock.mockResolvedValue({
    ok: true,
    json: async () => ({ givers: [], recipients: [] }),
  });
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
    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('start=2020-01-01');
    expect(url).toContain('end=2026-03-15');
  });
});

describe('Leaderboard quarter date range', () => {
  it('requests the local quarter boundaries', async () => {
    render(<Leaderboard year={2026} quarter={1} showRankChange={false} />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('start=2026-01-01');
    expect(url).toContain('end=2026-03-31');
  });
});
