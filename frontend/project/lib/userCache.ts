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

export function setCachedUsers(
  users: Record<string, { real_name: string; profile_image: string | null }>
): void {
  const cache = getUserCache();
  const now = Date.now();
  for (const [userId, data] of Object.entries(users)) {
    cache[userId] = {
      real_name: data.real_name,
      profile_image: data.profile_image,
      cached_at: now,
    };
  }
  setUserCache(cache);
}

/**
 * UserDataManager - Singleton for managing user data fetching with deduplication
 * Provides batch fetching, in-flight request deduplication, and localStorage caching
 */
class UserDataManager {
  private static instance: UserDataManager;
  private inFlightRequests: Map<string, Promise<CachedUserInfo | null>>;
  private readonly BATCH_SIZE = 100; // Max 100 IDs per batch request

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
    // Check localStorage cache first
    const cached = getCachedUser(userId);
    if (cached) {
      return cached;
    }

    // Check if request is already in flight
    const existing = this.inFlightRequests.get(userId);
    if (existing) {
      return existing;
    }

    // Create new request
    const promise = this.fetchSingleUser(userId);
    this.inFlightRequests.set(userId, promise);

    try {
      const result = await promise;
      return result;
    } finally {
      this.inFlightRequests.delete(userId);
    }
  }

  /**
   * Fetch multiple users efficiently with batching
   */
  async getUsers(userIds: string[]): Promise<Record<string, CachedUserInfo>> {
    const results: Record<string, CachedUserInfo> = {};
    const missing: string[] = [];

    // Check cache for all users
    for (const userId of userIds) {
      const cached = getCachedUser(userId);
      if (cached) {
        results[userId] = cached;
      } else {
        missing.push(userId);
      }
    }

    if (missing.length === 0) {
      return results;
    }

    // Fetch missing users in batches
    const batches: string[][] = [];
    for (let i = 0; i < missing.length; i += this.BATCH_SIZE) {
      batches.push(missing.slice(i, i + this.BATCH_SIZE));
    }

    // Fetch all batches in parallel, with retry on partial failure
    const batchResults = await Promise.allSettled(
      batches.map((batch) => this.fetchBatchUsers(batch))
    );

    // Collect results and identify failures
    const failedIds: string[] = [];
    for (let i = 0; i < batchResults.length; i++) {
      const result = batchResults[i];
      if (result.status === "fulfilled") {
        Object.assign(results, result.value);
        // Track which IDs from this batch failed
        const batchIds = batches[i];
        for (const id of batchIds) {
          if (!result.value[id]) {
            failedIds.push(id);
          }
        }
      } else {
        // Entire batch failed - add all IDs to retry
        failedIds.push(...batches[i]);
      }
    }

    // Retry failed IDs individually (partial render strategy)
    if (failedIds.length > 0 && failedIds.length <= 10) {
      // Only retry individually if few failures
      const retryResults = await Promise.allSettled(
        failedIds.map((id) => this.fetchSingleUser(id))
      );

      for (let i = 0; i < failedIds.length; i++) {
        const result = retryResults[i];
        if (result.status === "fulfilled" && result.value) {
          results[failedIds[i]] = result.value;
        }
      }
    }

    return results;
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

      // Cache to localStorage
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

      const data: Record<
        string,
        { real_name: string; profile_image: string }
      > = await response.json();

      // Convert to CachedUserInfo format
      const results: Record<string, CachedUserInfo> = {};
      const now = Date.now();

      for (const [userId, userData] of Object.entries(data)) {
        results[userId] = {
          real_name: userData.real_name,
          profile_image: userData.profile_image,
          cached_at: now,
        };
      }

      // Batch cache to localStorage
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
