import { getCachedUser, setCachedUsers, userDataManager } from "./userCache";
import type { PairStats, TopUserStats } from "./statsTypes";

/** Display data for one Slack user ID. */
export type ResolvedUser = { name: string; avatar: string | null };

/**
 * A user ID as it appears in a stats response, with the inline display data if the backend sent
 * it. `avatar` only counts next to a `name`; a missing avatar then means "no avatar".
 */
export type UserRef = { id: string; name?: string | null; avatar?: string | null };

type BatchLookup = Pick<typeof userDataManager, "getUsers">;

export function topUserRefs(rows: readonly TopUserStats[]): UserRef[] {
  return rows.map((r) => ({
    id: r.userId,
    name: r.real_name,
    avatar: r.profile_image,
  }));
}

export function pairUserRefs(rows: readonly PairStats[]): UserRef[] {
  return rows.flatMap((r) => [
    { id: r.giver, name: r.giver_name, avatar: r.giver_image },
    { id: r.recipient, name: r.recipient_name, avatar: r.recipient_image },
  ]);
}

/**
 * Names that are known right now, without a request: the inline name of a ref, else a cached
 * name. Lets a chart label new rows at once while resolveUsers runs.
 */
export function knownUserNames(refs: readonly UserRef[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const ref of refs) {
    if (!ref.id || out[ref.id]) continue;
    const name = ref.name || getCachedUser(ref.id)?.real_name;
    if (name) out[ref.id] = name;
  }
  return out;
}

/**
 * Names and avatars for the given user IDs.
 *
 * Display data the backend returned inline is used as it is (and stored in the local cache), so a
 * response that carries all names needs no further request. An inline name without an avatar
 * means the user has no avatar (the backend omits empty fields). Only IDs without an inline name
 * are looked up through the user cache / batch endpoint (userDataManager). Unknown IDs fall back
 * to the ID itself.
 */
export async function resolveUsers(
  refs: readonly UserRef[],
  lookup: BatchLookup = userDataManager
): Promise<Record<string, ResolvedUser>> {
  const out: Record<string, ResolvedUser> = {};
  const toStore: Record<string, { real_name: string; profile_image: string | null }> = {};

  for (const ref of refs) {
    if (!ref.id || out[ref.id] || !ref.name) continue;
    const avatar = ref.avatar || null;
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
