import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  CACHE_TTL_MS,
  MAX_CACHE_ENTRIES,
  flushUserCache,
  getCachedUser,
  resetUserCacheForTests,
  setCachedUser,
  setCachedUsers,
  userDataManager,
} from '../userCache';

const KEY = 'beerbot_user_cache';
const fetchMock = vi.fn();

function jsonResponse(body: unknown, ok = true) {
  return { ok, statusText: ok ? 'OK' : 'Bad Gateway', json: async () => body };
}

function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => { resolve = r; });
  return { promise, resolve };
}

/** Fake batch endpoint: returns a user for every requested ID except those in `missing`. */
function batchResponder(missing: string[] = []) {
  return async (url: string) => {
    const u = new URL(url, 'http://localhost');
    if (u.pathname === '/api/proxy/users') {
      const ids = (u.searchParams.get('ids') ?? '').split(',');
      const body = Object.fromEntries(
        ids.filter((id) => !missing.includes(id)).map((id) => [id, { real_name: `Name ${id}`, profile_image: null }])
      );
      return jsonResponse(body);
    }
    if (u.pathname === '/api/proxy/user') {
      const id = u.searchParams.get('user') ?? '';
      return missing.includes(id) ? jsonResponse({}, false) : jsonResponse({ real_name: `Single ${id}`, profile_image: null });
    }
    throw new Error(`unexpected ${url}`);
  };
}

beforeEach(() => {
  localStorage.clear();
  resetUserCacheForTests();
  userDataManager.clearInFlight();
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('local cache', () => {
  it('parses localStorage only once for many lookups (F4)', () => {
    const now = Date.now();
    localStorage.setItem(KEY, JSON.stringify({
      U1: { real_name: 'One', profile_image: null, cached_at: now },
      U2: { real_name: 'Two', profile_image: 'https://example.invalid/a.png', cached_at: now },
    }));
    resetUserCacheForTests();
    const getItem = vi.spyOn(Storage.prototype, 'getItem');

    for (let i = 0; i < 50; i++) {
      expect(getCachedUser('U1')?.real_name).toBe('One');
      expect(getCachedUser('U2')?.profile_image).toBe('https://example.invalid/a.png');
    }
    expect(getItem).toHaveBeenCalledTimes(1);
  });

  it('expires entries after the TTL', () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-05-01T12:00:00Z'));
    setCachedUser('U1', 'One', null);
    expect(getCachedUser('U1')).not.toBeNull();

    vi.setSystemTime(Date.now() + CACHE_TTL_MS + 1);
    expect(getCachedUser('U1')).toBeNull();
  });

  it('writes through to localStorage in the background and on flush', () => {
    setCachedUsers({ U1: { real_name: 'One', profile_image: null } });
    flushUserCache();
    const stored = JSON.parse(localStorage.getItem(KEY) ?? '{}');
    expect(stored.U1.real_name).toBe('One');

    // a fresh page load reads it back
    resetUserCacheForTests();
    expect(getCachedUser('U1')?.real_name).toBe('One');
  });

  it('merges what another tab stored instead of overwriting it', () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-05-01T12:00:00Z'));
    setCachedUser('U1', 'Mine', null);
    setCachedUser('U2', 'Old here', null);

    // another tab writes U3 and a newer U2 to the same key
    const later = Date.now() + 1000;
    localStorage.setItem(KEY, JSON.stringify({
      U2: { real_name: 'Newer there', profile_image: null, cached_at: later },
      U3: { real_name: 'Theirs', profile_image: null, cached_at: later },
    }));
    flushUserCache();

    const stored = JSON.parse(localStorage.getItem(KEY) ?? '{}');
    expect(Object.keys(stored).sort()).toEqual(['U1', 'U2', 'U3']);
    expect(stored.U2.real_name).toBe('Newer there');
    // this tab sees the other tab's entries without reloading
    expect(getCachedUser('U3')?.real_name).toBe('Theirs');

    // and an older entry from storage does not replace a newer one here
    setCachedUser('U1', 'Mine again', null);
    localStorage.setItem(KEY, JSON.stringify({ U1: { real_name: 'Stale', profile_image: null, cached_at: 1 } }));
    flushUserCache();
    expect(JSON.parse(localStorage.getItem(KEY) ?? '{}').U1.real_name).toBe('Mine again');
  });

  it('batches many writes into one localStorage write', async () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    for (let i = 0; i < 20; i++) setCachedUser(`U${i}`, `N${i}`, null);
    expect(setItem).not.toHaveBeenCalled();
    await new Promise((r) => setTimeout(r, 20));
    expect(setItem).toHaveBeenCalledTimes(1);
  });

  it(`keeps at most ${MAX_CACHE_ENTRIES} users, dropping the oldest writes`, () => {
    const users: Record<string, { real_name: string; profile_image: null }> = {};
    for (let i = 0; i < MAX_CACHE_ENTRIES + 25; i++) users[`U${i}`] = { real_name: `N${i}`, profile_image: null };
    setCachedUsers(users);
    flushUserCache();

    expect(Object.keys(JSON.parse(localStorage.getItem(KEY) ?? '{}'))).toHaveLength(MAX_CACHE_ENTRIES);
    expect(getCachedUser('U0')).toBeNull();
    expect(getCachedUser(`U${MAX_CACHE_ENTRIES + 24}`)?.real_name).toBe(`N${MAX_CACHE_ENTRIES + 24}`);
  });

  it('ignores a corrupt blob and invalid entries', () => {
    localStorage.setItem(KEY, '{not json');
    resetUserCacheForTests();
    expect(getCachedUser('U1')).toBeNull();

    localStorage.setItem(KEY, JSON.stringify({ U1: { real_name: 42 }, U2: { real_name: 'Two', profile_image: null, cached_at: Date.now() } }));
    resetUserCacheForTests();
    expect(getCachedUser('U1')).toBeNull();
    expect(getCachedUser('U2')?.real_name).toBe('Two');
  });

  it('works when localStorage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied'); });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('denied'); });
    resetUserCacheForTests();

    setCachedUser('U1', 'One', null);
    expect(() => flushUserCache()).not.toThrow();
    expect(getCachedUser('U1')?.real_name).toBe('One');
  });
});

describe('userDataManager.getUsers', () => {
  it('serves cached users without a request', async () => {
    setCachedUser('U1', 'One', null);
    const result = await userDataManager.getUsers(['U1']);
    expect(result.U1.real_name).toBe('One');
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('splits more than 100 IDs into batches of 100', async () => {
    fetchMock.mockImplementation(batchResponder());
    const ids = Array.from({ length: 150 }, (_, i) => `U${i}`);
    const result = await userDataManager.getUsers(ids);

    expect(Object.keys(result)).toHaveLength(150);
    const sizes = fetchMock.mock.calls.map(([url]) =>
      (new URL(String(url), 'http://localhost').searchParams.get('ids') ?? '').split(',').length
    );
    expect(sizes).toEqual([100, 50]);
  });

  it('shares in-flight lookups between concurrent callers (F3)', async () => {
    const gate = deferred<void>();
    const respond = batchResponder();
    fetchMock.mockImplementation(async (url: string) => {
      await gate.promise;
      return respond(url);
    });

    const a = userDataManager.getUsers(['U1', 'U2', 'U3']);
    const b = userDataManager.getUsers(['U2', 'U3', 'U4']);
    const c = userDataManager.getUser('U1');
    gate.resolve();
    const [ra, rb, rc] = await Promise.all([a, b, c]);

    expect(Object.keys(ra).sort()).toEqual(['U1', 'U2', 'U3']);
    expect(Object.keys(rb).sort()).toEqual(['U2', 'U3', 'U4']);
    expect(rc?.real_name).toBe('Name U1');
    // two batches: U1-U3 and only the new U4; no single-user request
    const requested = fetchMock.mock.calls.map(([url]) => new URL(String(url), 'http://localhost').searchParams.get('ids'));
    expect(requested).toEqual(['U1,U2,U3', 'U4']);
  });

  it('de-duplicates IDs within one call and skips empty IDs', async () => {
    fetchMock.mockImplementation(batchResponder());
    await userDataManager.getUsers(['U1', 'U1', '', ' ', 'U2']);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(String(fetchMock.mock.calls[0][0])).toContain(encodeURIComponent('U1,U2'));
  });

  it('retries a few missing IDs one by one', async () => {
    fetchMock.mockImplementation(batchResponder(['U2']));
    const result = await userDataManager.getUsers(['U1', 'U2']);

    expect(result.U1.real_name).toBe('Name U1');
    expect(result.U2).toBeUndefined(); // single lookup failed too
    const paths = fetchMock.mock.calls.map(([url]) => new URL(String(url), 'http://localhost').pathname);
    expect(paths).toEqual(['/api/proxy/users', '/api/proxy/user']);
  });

  it('retries one by one after a failed batch and does not retry many failures', async () => {
    fetchMock.mockImplementation(async (url: string) => {
      const u = new URL(url, 'http://localhost');
      if (u.pathname === '/api/proxy/users') return jsonResponse({}, false);
      return jsonResponse({ real_name: `Single ${u.searchParams.get('user')}`, profile_image: null });
    });
    const few = await userDataManager.getUsers(['U1', 'U2']);
    expect(few.U1.real_name).toBe('Single U1');

    fetchMock.mockClear();
    const many = Array.from({ length: 11 }, (_, i) => `X${i}`);
    expect(await userDataManager.getUsers(many)).toEqual({});
    expect(fetchMock).toHaveBeenCalledTimes(1); // only the batch
  });

  it('getUser caches the result of a single lookup', async () => {
    fetchMock.mockImplementation(batchResponder());
    expect((await userDataManager.getUser('U9'))?.real_name).toBe('Single U9');
    expect((await userDataManager.getUser('U9'))?.real_name).toBe('Single U9');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(await userDataManager.getUser('')).toBeNull();
  });

  it('does not remember a 503 (Slack still resolving): the next call asks again', async () => {
    fetchMock.mockResolvedValueOnce({ ...jsonResponse({}, false), status: 503, statusText: 'Service Unavailable' });
    expect(await userDataManager.getUser('U5')).toBeNull();
    expect(getCachedUser('U5')).toBeNull();

    fetchMock.mockImplementation(batchResponder());
    expect((await userDataManager.getUser('U5'))?.real_name).toBe('Single U5');
    expect(getCachedUser('U5')?.real_name).toBe('Single U5');
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
