import { beforeEach, describe, expect, it, vi } from 'vitest';
import { pairUserRefs, resolveUsers, topUserRefs } from '../userNames';
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

describe('resolveUsers', () => {
  it('uses inline names and makes no lookup when all are present', async () => {
    const lookup = lookupReturning({});
    const refs = topUserRefs([
      { userId: 'U1', count: 3, name: 'Alice Example', avatar: 'https://example.invalid/1.png' },
      { userId: 'U2', count: 1, name: 'Bob Example', avatar: null },
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
      { userId: 'U1', count: 3, name: 'Alice Example' },
      { userId: 'U2', count: 2 },
      { userId: 'U3', count: 1, name: '' },
    ]);

    const result = await resolveUsers(refs, lookup);

    expect(lookup.getUsers).toHaveBeenCalledWith(['U2', 'U3']);
    expect(result.U2).toEqual({ name: 'Bob Example', avatar: null });
    expect(result.U3).toEqual({ name: 'U3', avatar: null });
  });

  it('keeps a cached avatar when the inline entry has a name but no avatar field', async () => {
    setCachedUser('U1', 'Old Name', 'https://example.invalid/old.png');
    const result = await resolveUsers([{ id: 'U1', name: 'New Name' }], lookupReturning({}));
    expect(result.U1).toEqual({ name: 'New Name', avatar: 'https://example.invalid/old.png' });
  });

  it('takes an inline name from any occurrence of an ID', async () => {
    const lookup = lookupReturning({});
    const refs = pairUserRefs([
      { giver: 'U1', recipient: 'U2', count: 2 },
      { giver: 'U2', recipient: 'U1', count: 1, giverName: 'Bob Example', recipientName: 'Alice Example' },
    ]);

    const result = await resolveUsers(refs, lookup);

    expect(lookup.getUsers).not.toHaveBeenCalled();
    expect(result.U1.name).toBe('Alice Example');
    expect(result.U2.name).toBe('Bob Example');
  });
});
