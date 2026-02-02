"use client";

import { useEffect, useState, useRef } from "react";
import Image from "next/image";
import { RankChangeIndicator } from "./RankChangeIndicator";
import { getQuarterDates, getPreviousQuarter } from "@/lib/quarters";
import {
  getUserCache,
  getCachedUser,
  setCachedUser,
} from "@/lib/userCache";

interface LeaderboardProps {
  year?: number | null;
  quarter?: number | null;
  showRankChange?: boolean;
  dateRange?: { start: string; end: string } | null;
  maxHeight?: string;
}

type UserStats = {
  userId: string;
  count: number;
};

interface LeaderboardColumnProps {
  title: "Givers" | "Recipients";
  users: UserStats[];
  previousRanks: Record<string, number>;
  showRankChange: boolean;
  names: Record<string, string>;
  avatars: Record<string, string | null>;
  maxHeight?: string;
}

function LeaderboardColumn({
  title,
  users,
  previousRanks,
  showRankChange,
  names,
  avatars,
  maxHeight,
}: LeaderboardColumnProps) {
  const total = users.reduce((sum, u) => sum + u.count, 0);

  return (
    <div
      className="p-5 rounded-2xl shadow-lg min-h-[400px] border"
      style={{
        backgroundColor: "hsl(var(--card))",
        color: "hsl(var(--card-foreground))",
        borderColor: "hsl(var(--border))",
      }}
    >
      <h2
        className="text-xl font-bold mb-2 flex items-center gap-2"
        style={{ color: "hsl(var(--primary))" }}
      >
        {title === "Givers" ? "\u{1F64C}" : "\u{1F389}"} {title}
      </h2>
      <div
        className="mb-3 text-sm"
        style={{ color: "hsl(var(--muted-foreground))" }}
      >
        Total{" "}
        <span
          className="font-bold"
          style={{ color: "hsl(var(--primary))" }}
        >
          {"\u{1F37A}"} {total}
        </span>
      </div>
      {users.length === 0 ? (
        <div className="text-sm" style={{ color: "hsl(var(--muted-foreground))" }}>
          No data
        </div>
      ) : (
        <div className={maxHeight ? `max-h-[${maxHeight}] overflow-y-auto pr-1` : "pr-1"} style={maxHeight ? { maxHeight, overflowY: "auto" } : undefined}>
          <ul className="divide-y" style={{ borderColor: "hsl(var(--border))" }}>
            {users.map(({ userId, count }, i) => {
              const rank = i + 1;
              const previousRank = showRankChange
                ? previousRanks[userId] ?? null
                : null;

              return (
                <li
                  key={userId}
                  className="flex items-center justify-between py-2 group transition rounded-lg px-2 hover:opacity-80"
                  style={{
                    animation: `fadein 0.3s ${i * 0.01}s both`,
                    borderColor: "hsl(var(--border))",
                  }}
                >
                  <div className="flex items-center gap-3">
                    {avatars[userId] ? (
                      <Image
                        src={avatars[userId] as string}
                        alt={names[userId] || userId}
                        width={32}
                        height={32}
                        unoptimized
                        className="w-8 h-8 rounded-full object-cover shadow-sm border"
                        style={{ borderColor: "hsl(var(--border))" }}
                      />
                    ) : (
                      <span
                        className="inline-flex items-center justify-center w-8 h-8 rounded-full font-bold text-lg shadow-sm"
                        style={{
                          backgroundColor: "hsl(var(--accent))",
                          color: "hsl(var(--accent-foreground))",
                        }}
                      >
                        {names[userId]?.[0]?.toUpperCase() || userId[0]}
                      </span>
                    )}
                    <span
                      className="text-base font-medium"
                      style={{ color: "hsl(var(--foreground))" }}
                    >
                      {names[userId] || userId}
                    </span>
                    {showRankChange && (
                      <RankChangeIndicator
                        currentRank={rank}
                        previousRank={previousRank}
                      />
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-lg">{"\u{1F37A}"}</span>
                    <span
                      className="text-base font-bold"
                      style={{ color: "hsl(var(--primary))" }}
                    >
                      {count}
                    </span>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
      {[0, 1].map((col) => (
        <div
          key={col}
          className="p-5 rounded-2xl shadow-lg min-h-[400px] border animate-pulse"
          style={{
            backgroundColor: "hsl(var(--card))",
            borderColor: "hsl(var(--border))",
          }}
        >
          <div
            className="h-7 w-32 rounded mb-4"
            style={{ backgroundColor: "hsl(var(--muted))" }}
          />
          <div
            className="h-4 w-24 rounded mb-4"
            style={{ backgroundColor: "hsl(var(--muted))" }}
          />
          {[0, 1, 2, 3].map((row) => (
            <div
              key={row}
              className="flex items-center gap-3 py-3"
            >
              <div
                className="w-8 h-8 rounded-full"
                style={{ backgroundColor: "hsl(var(--muted))" }}
              />
              <div
                className="h-4 w-32 rounded"
                style={{ backgroundColor: "hsl(var(--muted))" }}
              />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

export function Leaderboard({
  year = null,
  quarter = null,
  showRankChange = true,
  dateRange = null,
  maxHeight = "660px",
}: LeaderboardProps) {
  const [givers, setGivers] = useState<UserStats[]>([]);
  const [recipients, setRecipients] = useState<UserStats[]>([]);
  const [prevGiverRanks, setPrevGiverRanks] = useState<Record<string, number>>({});
  const [prevRecipientRanks, setPrevRecipientRanks] = useState<Record<string, number>>({});
  const [names, setNames] = useState<Record<string, string>>({});
  const [avatars, setAvatars] = useState<Record<string, string | null>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mounted = useRef(true);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      setLoading(true);
      setError(null);

      try {
        // Fetch all users first
        const [giversResp, recipientsResp] = await Promise.all([
          fetch("/api/proxy/givers"),
          fetch("/api/proxy/recipients"),
        ]);

        if (!giversResp.ok || !recipientsResp.ok) {
          throw new Error("Failed to fetch users");
        }

        const allGivers: string[] = await giversResp.json();
        const allRecipients: string[] = await recipientsResp.json();

        if (cancelled) return;

        // Calculate date range for current period
        let dateParams = "";
        if (dateRange) {
          // Use provided date range (homepage mode)
          dateParams = `&start=${dateRange.start}&end=${dateRange.end}`;
        } else if (year !== null && quarter !== null) {
          const { start, end } = getQuarterDates(year, quarter);
          dateParams = `&start=${start}&end=${end}`;
        } else {
          // All-time view: use wide date range (backend requires date params)
          const today = new Date().toISOString().split("T")[0];
          dateParams = `&start=2020-01-01&end=${today}`;
        }

        // Fetch stats for current period
        async function fetchStats(
          users: string[],
          type: "Givers" | "Recipients",
          params: string
        ): Promise<UserStats[]> {
          const path = type === "Givers" ? "/api/proxy/given" : "/api/proxy/received";
          const results: UserStats[] = [];

          const concurrency = 5;
          let idx = 0;

          async function worker() {
            while (idx < users.length && !cancelled) {
              const i = idx++;
              const userId = users[i];
              try {
                const resp = await fetch(`${path}?user=${encodeURIComponent(userId)}${params}`);
                if (!resp.ok) {
                  results.push({ userId, count: 0 });
                  continue;
                }
                const data = await resp.json();
                const count = type === "Givers" ? data.given : data.received;
                results.push({ userId, count });
              } catch {
                results.push({ userId, count: 0 });
              }
            }
          }

          const workers: Promise<void>[] = [];
          for (let i = 0; i < concurrency; i++) {
            workers.push(worker());
          }
          await Promise.all(workers);

          // Sort by count descending, filter out zeros, limit to top 100
          return results
            .sort((a, b) => b.count - a.count)
            .filter((u) => u.count > 0)
            .slice(0, 100);
        }

        const [currentGivers, currentRecipients] = await Promise.all([
          fetchStats(allGivers, "Givers", dateParams),
          fetchStats(allRecipients, "Recipients", dateParams),
        ]);

        if (cancelled) return;

        // If showing rank changes, fetch previous quarter data
        let prevGivers: Record<string, number> = {};
        let prevRecipients: Record<string, number> = {};

        if (showRankChange && year !== null && quarter !== null) {
          const prev = getPreviousQuarter(year, quarter);
          const { start: prevStart, end: prevEnd } = getQuarterDates(prev.year, prev.quarter);
          const prevParams = `&start=${prevStart}&end=${prevEnd}`;

          const [prevGiverStats, prevRecipientStats] = await Promise.all([
            fetchStats(allGivers, "Givers", prevParams),
            fetchStats(allRecipients, "Recipients", prevParams),
          ]);

          if (cancelled) return;

          // Build rank maps (1-indexed)
          prevGivers = Object.fromEntries(
            prevGiverStats.map((u, i) => [u.userId, i + 1])
          );
          prevRecipients = Object.fromEntries(
            prevRecipientStats.map((u, i) => [u.userId, i + 1])
          );
        }

        if (cancelled || !mounted.current) return;

        setGivers(currentGivers);
        setRecipients(currentRecipients);
        setPrevGiverRanks(prevGivers);
        setPrevRecipientRanks(prevRecipients);

        // Fetch names and avatars for all unique users
        const allUsers = new Set([
          ...currentGivers.map((u) => u.userId),
          ...currentRecipients.map((u) => u.userId),
        ]);

        const namesOut: Record<string, string> = {};
        const avatarsOut: Record<string, string | null> = {};

        const userList = Array.from(allUsers);
        let userIdx = 0;

        async function fetchUserInfo() {
          while (userIdx < userList.length && !cancelled) {
            const i = userIdx++;
            const userId = userList[i];

            // Check cache first
            const cached = getCachedUser(userId);
            if (cached) {
              namesOut[userId] = cached.real_name;
              avatarsOut[userId] = cached.profile_image;
              continue;
            }

            try {
              const resp = await fetch(`/api/proxy/user?user=${encodeURIComponent(userId)}`);
              if (!resp.ok) {
                const expiredCache = getUserCache()[userId];
                if (expiredCache) {
                  namesOut[userId] = expiredCache.real_name;
                  avatarsOut[userId] = expiredCache.profile_image;
                } else {
                  namesOut[userId] = userId;
                  avatarsOut[userId] = null;
                }
                continue;
              }
              const data = await resp.json();
              const realName = data.real_name || userId;
              const profileImage = data.profile_image || null;
              namesOut[userId] = realName;
              avatarsOut[userId] = profileImage;
              if (data.real_name) {
                setCachedUser(userId, realName, profileImage);
              }
            } catch {
              const expiredCache = getUserCache()[userId];
              if (expiredCache) {
                namesOut[userId] = expiredCache.real_name;
                avatarsOut[userId] = expiredCache.profile_image;
              } else {
                namesOut[userId] = userId;
                avatarsOut[userId] = null;
              }
            }
          }
        }

        const userWorkers: Promise<void>[] = [];
        for (let i = 0; i < 5; i++) {
          userWorkers.push(fetchUserInfo());
        }
        await Promise.all(userWorkers);

        if (cancelled || !mounted.current) return;

        setNames(namesOut);
        setAvatars(avatarsOut);
        setLoading(false);
      } catch (err) {
        if (cancelled || !mounted.current) return;
        setError(err instanceof Error ? err.message : "An error occurred");
        setLoading(false);
      }
    }

    fetchData();

    return () => {
      cancelled = true;
    };
  }, [year, quarter, showRankChange, dateRange]);

  if (loading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <div
        className="p-6 rounded-2xl border text-center"
        style={{
          backgroundColor: "hsl(var(--card))",
          borderColor: "hsl(var(--border))",
        }}
      >
        <p className="text-lg mb-4" style={{ color: "hsl(var(--destructive))" }}>
          {error}
        </p>
        <button
          onClick={() => window.location.reload()}
          className="px-4 py-2 rounded-md text-sm font-semibold transition-colors"
          style={{
            backgroundColor: "hsl(var(--primary))",
            color: "hsl(var(--primary-foreground))",
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  if (givers.length === 0 && recipients.length === 0) {
    return (
      <div
        className="p-6 rounded-2xl border text-center"
        style={{
          backgroundColor: "hsl(var(--card))",
          borderColor: "hsl(var(--border))",
        }}
      >
        <p className="text-lg" style={{ color: "hsl(var(--muted-foreground))" }}>
          {"\u{1F37A}"} No beers recorded for this {year !== null && quarter !== null ? "quarter" : "period"}
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
      <LeaderboardColumn
        title="Givers"
        users={givers}
        previousRanks={prevGiverRanks}
        showRankChange={showRankChange && year !== null && quarter !== null}
        names={names}
        avatars={avatars}
        maxHeight={maxHeight}
      />
      <LeaderboardColumn
        title="Recipients"
        users={recipients}
        previousRanks={prevRecipientRanks}
        showRankChange={showRankChange && year !== null && quarter !== null}
        names={names}
        avatars={avatars}
        maxHeight={maxHeight}
      />
    </div>
  );
}
