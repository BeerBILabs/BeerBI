import { getCachedUser, setCachedUsers, userDataManager } from "./userCache";
import type { PairStats, TopUserStats } from "./statsTypes";

/** Display data for one Slack user ID. */
export type ResolvedUser = { name: string; avatar: string | null };

/** A user ID as it appears in a stats response, with the inline name/avatar if the backend sent one. */
export type UserRef = { id: string; name?: string | null; avatar?: string | null };

type BatchLookup = Pick<typeof userDataManager, "getUsers">;

export function topUserRefs(rows: readonly TopUserStats[]): UserRef[] {
  return rows.map((r) => ({ id: r.userId, name: r.name, avatar: r.avatar }));
}

export function pairUserRefs(rows: readonly PairStats[]): UserRef[] {
  return rows.flatMap((r) => [
    { id: r.giver, name: r.giverName },
    { id: r.recipient, name: r.recipientName },
  ]);
}

/**
 * Names and avatars for the given user IDs.
 *
 * Names the backend returned inline are used as they are (and stored in the local cache), so a
 * response that carries all names needs no further request. Only IDs without an inline name are
 * looked up through the user cache / batch endpoint (userDataManager). Unknown IDs fall back to
 * the ID itself.
 */
export async function resolveUsers(
  refs: readonly UserRef[],
  lookup: BatchLookup = userDataManager
): Promise<Record<string, ResolvedUser>> {
  const out: Record<string, ResolvedUser> = {};
  const toStore: Record<string, { real_name: string; profile_image: string | null }> = {};

  for (const ref of refs) {
    if (!ref.id || out[ref.id] || !ref.name) continue;
    // An inline name without an avatar field keeps a cached avatar instead of dropping it.
    const avatar =
      ref.avatar !== undefined ? ref.avatar : (getCachedUser(ref.id)?.profile_image ?? null);
    out[ref.id] = { name: ref.name, avatar };
    toStore[ref.id] = { real_name: ref.name, profile_image: avatar };
  }
  if (Object.keys(toStore).length > 0) setCachedUsers(toStore);

  const missing = Array.from(new Set(refs.map((r) => r.id))).filter((id) => id && !out[id]);
  if (missing.length > 0) {
    const infos = await lookup.getUsers(missing);
    for (const id of missing) {
      const info = infos[id];
      out[id] = info
        ? { name: info.real_name || id, avatar: info.profile_image ?? null }
        : { name: id, avatar: null };
    }
  }

  return out;
}
