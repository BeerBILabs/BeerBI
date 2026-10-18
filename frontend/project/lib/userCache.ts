export type CachedUserInfo = {
  real_name: string;
  profile_image: string | null;
  cached_at: number;
};

type UserData = { real_name: string; profile_image: string | null };

const USER_CACHE_KEY = "beerbot_user_cache";
export const CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000; // 7 days
/** Upper bound for stored users; the entries written longest ago are dropped first. */
export const MAX_CACHE_ENTRIES = 500;

// The localStorage blob is parsed once into this Map and written back in the background
// (audit F4: it used to be parsed for every single lookup). Map order is write order, so the
// first entries are the ones written longest ago.
let memory: Map<string, CachedUserInfo> | null = null;
let flushScheduled = false;
let flushGeneration = 0; // bumped by the test reset so a pending flush of an old cache is dropped
let pagehideHooked = false;

function getStorage(): Storage | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage;
  } catch {
    return null; // access can throw when storage is disabled
  }
}

function isCachedUserInfo(value: unknown): value is CachedUserInfo {
  if (!value || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    typeof v.real_name === "string" &&
    typeof v.cached_at === "number" &&
    (v.profile_image === null || typeof v.profile_image === "string")
  );
}

function trim(map: Map<string, CachedUserInfo>): void {
  while (map.size > MAX_CACHE_ENTRIES) {
    const oldest = map.keys().next().value;
    if (oldest === undefined) break;
    map.delete(oldest);
  }
}

function load(): Map<string, CachedUserInfo> {
  // On the server there is no storage and nothing must be shared between requests.
  if (typeof window === "undefined") return new Map();
  if (memory) return memory;

  memory = new Map();
  const storage = getStorage();
  if (storage) {
    try {
      const raw = storage.getItem(USER_CACHE_KEY);
      const parsed: unknown = raw ? JSON.parse(raw) : null;
      if (parsed && typeof parsed === "object") {
        const entries = Object.entries(parsed as Record<string, unknown>)
          .filter((e): e is [string, CachedUserInfo] => isCachedUserInfo(e[1]))
          .sort((a, b) => a[1].cached_at - b[1].cached_at);
        for (const [id, info] of entries) memory.set(id, info);
        trim(memory);
      }
    } catch {
      // corrupt blob: start empty, the next flush overwrites it
    }
  }

  if (!pagehideHooked) {
    pagehideHooked = true;
    window.addEventListener("pagehide", flushUserCache);
  }
  return memory;
}

/** Write the in-memory cache to localStorage now. */
export function flushUserCache(): void {
  flushScheduled = false;
  const storage = getStorage();
  if (!storage || !memory) return;
  try {
    storage.setItem(USER_CACHE_KEY, JSON.stringify(Object.fromEntries(memory)));
  } catch {
    // localStorage might be full or disabled
  }
}

function scheduleFlush(): void {
  if (flushScheduled || typeof window === "undefined") return;
  flushScheduled = true;
  const generation = flushGeneration;
  const run = () => {
    if (generation === flushGeneration) flushUserCache();
  };
  if (typeof window.requestIdleCallback === "function") {
    window.requestIdleCallback(run, { timeout: 1000 });
  } else {
    setTimeout(run, 0);
  }
}

export function getCachedUser(userId: string): CachedUserInfo | null {
  const entry = load().get(userId);
  if (!entry) return null;
  if (Date.now() - entry.cached_at > CACHE_TTL_MS) return null;
  return entry;
}

export function setCachedUsers(users: Record<string, UserData>): void {
  const map = load();
  const now = Date.now();
  for (const [userId, data] of Object.entries(users)) {
    map.delete(userId); // re-insert at the end: most recently written
    map.set(userId, {
      real_name: data.real_name,
      profile_image: data.profile_image,
      cached_at: now,
    });
  }
  trim(map);
  scheduleFlush();
}

export function setCachedUser(
  userId: string,
  realName: string,
  profileImage: string | null
): void {
  setCachedUsers({ [userId]: { real_name: realName, profile_image: profileImage } });
}

/** Test helper: forget the in-memory copy so the next access reads localStorage again. */
export function resetUserCacheForTests(): void {
  flushGeneration++;
  memory = null;
  flushScheduled = false;
}

/**
 * UserDataManager - Singleton for managing user data fetching with deduplication
 * Provides batch fetching, in-flight request deduplication (per user ID, shared by single and
 * batch lookups) and caching.
 */
class UserDataManager {
  private static instance: UserDataManager;
  private inFlightRequests: Map<string, Promise<CachedUserInfo | null>>;
  private readonly BATCH_SIZE = 100; // Max 100 IDs per batch request
  private readonly MAX_INDIVIDUAL_RETRIES = 10; // retry missing IDs one by one only if few failed

  private constructor() {
    this.inFlightRequests = new Map();
  }

  static getInstance(): UserDataManager {
    if (!UserDataManager.instance) {
      UserDataManager.instance = new UserDataManager();
    }
    return UserDataManager.instance;
  }

  /**
   * Fetch a single user, with deduplication and caching
   */
  async getUser(userId: string): Promise<CachedUserInfo | null> {
    if (!userId || userId.trim() === "") {
      return null;
    }

    const cached = getCachedUser(userId);
    if (cached) {
      return cached;
    }

    return (
      this.inFlightRequests.get(userId) ??
      this.track(userId, this.fetchSingleUser(userId))
    );
  }

  /**
   * Fetch multiple users efficiently with batching. IDs that another call is already
   * fetching are not requested again; their pending result is shared.
   */
  async getUsers(userIds: string[]): Promise<Record<string, CachedUserInfo>> {
    const results: Record<string, CachedUserInfo> = {};
    const uniqueIds = Array.from(
      new Set(userIds.filter((id) => id && id.trim() !== ""))
    );

    const pending: Array<[string, Promise<CachedUserInfo | null>]> = [];
    const toFetch: string[] = [];

    for (const userId of uniqueIds) {
      const cached = getCachedUser(userId);
      if (cached) {
        results[userId] = cached;
        continue;
      }
      const inFlight = this.inFlightRequests.get(userId);
      if (inFlight) {
        pending.push([userId, inFlight]);
      } else {
        toFetch.push(userId);
      }
    }

    for (let i = 0; i < toFetch.length; i += this.BATCH_SIZE) {
      const batch = toFetch.slice(i, i + this.BATCH_SIZE);
      const batchPromise = this.fetchBatchWithRetry(batch);
      for (const userId of batch) {
        pending.push([
          userId,
          this.track(userId, batchPromise.then((found) => found[userId] ?? null)),
        ]);
      }
    }

    const resolved = await Promise.all(pending.map(([, promise]) => promise));
    pending.forEach(([userId], i) => {
      const info = resolved[i];
      if (info) results[userId] = info;
    });

    return results;
  }

  /** Register a pending lookup so concurrent callers share it; removed once settled. */
  private track(
    userId: string,
    request: Promise<CachedUserInfo | null>
  ): Promise<CachedUserInfo | null> {
    const tracked: Promise<CachedUserInfo | null> = request
      .catch(() => null)
      .finally(() => {
        if (this.inFlightRequests.get(userId) === tracked) {
          this.inFlightRequests.delete(userId);
        }
      });
    this.inFlightRequests.set(userId, tracked);
    return tracked;
  }

  /** One batch request; IDs it did not return are retried one by one if only a few are missing. */
  private async fetchBatchWithRetry(
    userIds: string[]
  ): Promise<Record<string, CachedUserInfo>> {
    let found: Record<string, CachedUserInfo> = {};
    try {
      found = await this.fetchBatchUsers(userIds);
    } catch {
      found = {};
    }

    const failed = userIds.filter((id) => !found[id]);
    if (failed.length > 0 && failed.length <= this.MAX_INDIVIDUAL_RETRIES) {
      const retried = await Promise.all(failed.map((id) => this.fetchSingleUser(id)));
      failed.forEach((id, i) => {
        const info = retried[i];
        if (info) found[id] = info;
      });
    }
    return found;
  }

  /**
   * Fetch a single user from the backend
   */
  private async fetchSingleUser(
    userId: string
  ): Promise<CachedUserInfo | null> {
    try {
      const response = await fetch(
        `/api/proxy/user?user=${encodeURIComponent(userId)}`
      );

      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      const userInfo: CachedUserInfo = {
        real_name: data.real_name,
        profile_image: data.profile_image,
        cached_at: Date.now(),
      };

      setCachedUser(userId, data.real_name, data.profile_image);

      return userInfo;
    } catch (error) {
      console.error(`Failed to fetch user ${userId}:`, error);
      return null;
    }
  }

  /**
   * Fetch multiple users in a single batch request
   */
  private async fetchBatchUsers(
    userIds: string[]
  ): Promise<Record<string, CachedUserInfo>> {
    if (userIds.length === 0) {
      return {};
    }

    try {
      const idsParam = userIds.join(",");
      const response = await fetch(
        `/api/proxy/users?ids=${encodeURIComponent(idsParam)}`
      );

      if (!response.ok) {
        throw new Error(`Batch fetch failed: ${response.statusText}`);
      }

      const data: Record<string, UserData> = await response.json();

      const results: Record<string, CachedUserInfo> = {};
      const now = Date.now();

      for (const [userId, userData] of Object.entries(data)) {
        results[userId] = {
          real_name: userData.real_name,
          profile_image: userData.profile_image,
          cached_at: now,
        };
      }

      setCachedUsers(data);

      return results;
    } catch (error) {
      console.error("Batch user fetch failed:", error);
      throw error;
    }
  }

  /**
   * Clear all in-flight requests (useful for testing/cleanup)
   */
  clearInFlight(): void {
    this.inFlightRequests.clear();
  }
}

// Export singleton instance
export const userDataManager = UserDataManager.getInstance();
