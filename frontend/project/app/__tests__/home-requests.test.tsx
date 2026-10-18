import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import Home from '../page';
import { resetUserCacheForTests, userDataManager } from '@/lib/userCache';

// Audit §5: the first paint of `/` issues at most two backend requests (current period and
// comparison period, one `/api/leaderboard` call per leaderboard) and no `/users` lookups
// when the backend sends the names inline.

const fetchMock = vi.fn();

function urls(): string[] {
  return fetchMock.mock.calls.map(([url]) => String(url));
}

beforeEach(() => {
  localStorage.clear();
  resetUserCacheForTests();
  userDataManager.clearInFlight();
  fetchMock.mockReset();
  fetchMock.mockImplementation(async () => ({
    ok: true,
    statusText: 'OK',
    json: async () => ({
      current: {
        givers: [{ userId: 'U1', count: 5, real_name: 'Alice Example' }],
        recipients: [{ userId: 'U2', count: 4, real_name: 'Bob Example' }],
      },
      previous: { givers: [{ userId: 'U1', count: 1 }], recipients: [] },
    }),
  }));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe('first paint of /', () => {
  it('makes two /api/leaderboard requests and no /users request', async () => {
    render(<Home />);

    await waitFor(() => expect(screen.getAllByText('Alice Example')).toHaveLength(2));
    expect(screen.getAllByText('Bob Example')).toHaveLength(2);

    const calls = urls();
    expect(calls).toHaveLength(2);
    for (const url of calls) expect(url).toMatch(/^\/api\/leaderboard\?/);
    expect(calls.some((url) => url.includes('/users'))).toBe(false);
  });
});
