export type CachedUserInfo = {
  real_name: string;
  profile_image: string | null;
  cached_at: number;
};

const USER_CACHE_KEY = "beerbot_user_cache";
const CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000; // 7 days

export function getUserCache(): Record<string, CachedUserInfo> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(USER_CACHE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

export function setUserCache(cache: Record<string, CachedUserInfo>): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(USER_CACHE_KEY, JSON.stringify(cache));
  } catch {
    // localStorage might be full or disabled
  }
}

export function getCachedUser(userId: string): CachedUserInfo | null {
  const cache = getUserCache();
  const entry = cache[userId];
  if (!entry) return null;
  if (Date.now() - entry.cached_at > CACHE_TTL_MS) return null;
  return entry;
}

export function setCachedUser(
  userId: string,
  realName: string,
  profileImage: string | null
): void {
  const cache = getUserCache();
  cache[userId] = {
    real_name: realName,
    profile_image: profileImage,
    cached_at: Date.now(),
  };
  setUserCache(cache);
}
