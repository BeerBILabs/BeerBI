import { beforeEach, describe, expect, it, vi } from 'vitest';
import { knownUserNames, pairUserRefs, resolveUsers, topUserRefs } from '../userNames';
import { getCachedUser, resetUserCacheForTests, setCachedUser } from '../userCache';

function lookupReturning(users: Record<string, { real_name: string; profile_image: string | null }>) {
  return {
    getUsers: vi.fn(async (ids: string[]) =>
      Object.fromEntries(
        ids.filter((id) => users[id]).map((id) => [id, { ...users[id], cached_at: Date.now() }])
      )
    ),
  };
}

beforeEach(() => {
  localStorage.clear();
  resetUserCacheForTests();
});

describe('user refs', () => {
  it('reads the backend fields of top rows and pairs (stats_names.go)', () => {
    expect(
      topUserRefs([{ userId: 'U1', count: 1, real_name: 'Alice Example', profile_image: 'https://example.invalid/1.png' }])
    ).toEqual([{ id: 'U1', name: 'Alice Example', avatar: 'https://example.invalid/1.png' }]);
    expect(
      pairUserRefs([
        {
          giver: 'U1',
          recipient: 'U2',
          count: 1,
          giver_name: 'Alice Example',
          giver_image: 'https://example.invalid/1.png',
          recipient_name: 'Bob Example',
        },
      ])
    ).toEqual([
      { id: 'U1', name: 'Alice Example', avatar: 'https://example.invalid/1.png' },
      { id: 'U2', name: 'Bob Example', avatar: undefined },
    ]);
  });

  it('knownUserNames takes inline names, then cached names, and skips the rest', () => {
    setCachedUser('U2', 'Bob Example', null);
    expect(knownUserNames([{ id: 'U1', name: 'Alice Example' }, { id: 'U2' }, { id: 'U3' }, { id: '' }])).toEqual({
      U1: 'Alice Example',
      U2: 'Bob Example',
    });
  });
});

describe('resolveUsers', () => {
  it('uses inline names and makes no lookup when all are present', async () => {
    const lookup = lookupReturning({});
    const refs = topUserRefs([
      { userId: 'U1', count: 3, real_name: 'Alice Example', profile_image: 'https://example.invalid/1.png' },
      { userId: 'U2', count: 1, real_name: 'Bob Example' },
    ]);

    const result = await resolveUsers(refs, lookup);

    expect(lookup.getUsers).not.toHaveBeenCalled();
    expect(result).toEqual({
      U1: { name: 'Alice Example', avatar: 'https://example.invalid/1.png' },
      U2: { name: 'Bob Example', avatar: null },
    });
    // inline names also fill the local cache
    expect(getCachedUser('U1')?.real_name).toBe('Alice Example');
  });

  it('looks up only the IDs without an inline name and falls back to the ID', async () => {
    const lookup = lookupReturning({ U2: { real_name: 'Bob Example', profile_image: null } });
    const refs = topUserRefs([
      { userId: 'U1', count: 3, real_name: 'Alice Example' },
      { userId: 'U2', count: 2 },
      { userId: 'U3', count: 1, real_name: '' },
    ]);

    const result = await resolveUsers(refs, lookup);

    expect(lookup.getUsers).toHaveBeenCalledWith(['U2', 'U3']);
    expect(result.U2).toEqual({ name: 'Bob Example', avatar: null });
    expect(result.U3).toEqual({ name: 'U3', avatar: null });
  });

  it('treats a name without an image field as "no avatar" and replaces a cached avatar', async () => {
    // The backend omits empty fields: the user removed the avatar since it was cached.
    setCachedUser('U1', 'Old Name', 'https://example.invalid/old.png');
    const result = await resolveUsers(topUserRefs([{ userId: 'U1', count: 1, real_name: 'New Name' }]), lookupReturning({}));
    expect(result.U1).toEqual({ name: 'New Name', avatar: null });
    expect(getCachedUser('U1')).toMatchObject({ real_name: 'New Name', profile_image: null });
  });

  it('caches pair avatars, so a later lookup gets the image without a request', async () => {
    const refs = pairUserRefs([
      {
        giver: 'U1',
        recipient: 'U2',
        count: 2,
        giver_name: 'Alice Example',
        giver_image: 'https://example.invalid/1.png',
        recipient_name: 'Bob Example',
        recipient_image: 'https://example.invalid/2.png',
      },
    ]);
    await resolveUsers(refs, lookupReturning({}));
    expect(getCachedUser('U1')?.profile_image).toBe('https://example.invalid/1.png');
    expect(getCachedUser('U2')?.profile_image).toBe('https://example.invalid/2.png');
  });

  it('takes an inline name from any occurrence of an ID', async () => {
    const lookup = lookupReturning({});
    const refs = pairUserRefs([
      { giver: 'U1', recipient: 'U2', count: 2 },
      { giver: 'U2', recipient: 'U1', count: 1, giver_name: 'Bob Example', recipient_name: 'Alice Example' },
    ]);

    const result = await resolveUsers(refs, lookup);

    expect(lookup.getUsers).not.toHaveBeenCalled();
    expect(result.U1.name).toBe('Alice Example');
    expect(result.U2.name).toBe('Bob Example');
  });
});
