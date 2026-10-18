"use client";

import Image from "next/image";
import { RankChangeIndicator } from "./RankChangeIndicator";
import { getQuarterDates, getPreviousQuarter } from "@/lib/quarters";
import { formatLocalDate } from "@/lib/dateUtils";
import { resolveUsers, topUserRefs } from "@/lib/userNames";
import { useLatestRequest } from "@/lib/useLatestRequest";
import type { LeaderboardResponse, TopUserStats } from "@/lib/statsTypes";

interface LeaderboardProps {
  year?: number | null;
  quarter?: number | null;
  showRankChange?: boolean;
  dateRange?: { start: string; end: string } | null;
  maxHeight?: string;
}

type UserStats = TopUserStats;

type LeaderboardData = {
  givers: UserStats[];
  recipients: UserStats[];
  prevGiverRanks: Record<string, number>;
  prevRecipientRanks: Record<string, number>;
  names: Record<string, string>;
  avatars: Record<string, string | null>;
};

const ALL_TIME_START = "2020-01-01";
const TOP_LIMIT = 100;

/**
 * URL of the one request a leaderboard needs: /api/leaderboard returns the ranking of the
 * period and, in quarter mode with rank changes, of the previous quarter (fetched in parallel
 * on the server).
 */
export function leaderboardUrl({
  year,
  quarter,
  showRankChange,
  dateRange,
}: Pick<LeaderboardProps, "year" | "quarter" | "showRankChange" | "dateRange">): string {
  const params = new URLSearchParams({ limit: String(TOP_LIMIT) });
  if (dateRange) {
    params.set("start", dateRange.start);
    params.set("end", dateRange.end);
  } else if (year != null && quarter != null) {
    const { start, end } = getQuarterDates(year, quarter);
    params.set("start", start);
    params.set("end", end);
    if (showRankChange) {
      const prev = getPreviousQuarter(year, quarter);
      const prevDates = getQuarterDates(prev.year, prev.quarter);
      params.set("prev_start", prevDates.start);
      params.set("prev_end", prevDates.end);
    }
  } else {
    // All-time view: wide date range (the backend requires date params)
    params.set("start", ALL_TIME_START);
    params.set("end", formatLocalDate(new Date()));
  }
  return `/api/leaderboard?${params}`;
}

function rankMap(users: readonly UserStats[]): Record<string, number> {
  return Object.fromEntries(users.map((u, i) => [u.userId, i + 1]));
}

async function loadLeaderboard(url: string, signal: AbortSignal): Promise<LeaderboardData> {
  const resp = await fetch(url, { signal });
  if (!resp.ok) {
    throw new Error("Failed to fetch top users");
  }
  const body: Partial<LeaderboardResponse> = await resp.json();
  const givers = body.current?.givers ?? [];
  const recipients = body.current?.recipients ?? [];

  // Names from the stats response when the backend sends them inline, else one batch lookup.
  const users = await resolveUsers([...topUserRefs(givers), ...topUserRefs(recipients)]);
  const names: Record<string, string> = {};
  const avatars: Record<string, string | null> = {};
  for (const [id, u] of Object.entries(users)) {
    names[id] = u.name;
    avatars[id] = u.avatar;
  }

  return {
    givers,
    recipients,
    prevGiverRanks: rankMap(body.previous?.givers ?? []),
    prevRecipientRanks: rankMap(body.previous?.recipients ?? []),
    names,
    avatars,
  };
}

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
  const url = leaderboardUrl({ year, quarter, showRankChange, dateRange });
  const { data, isFetching, error } = useLatestRequest(url, loadLeaderboard);

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

  if (!data) {
    return <LoadingSkeleton />;
  }

  // While another range loads, the previous ranking stays visible (dimmed) instead of a skeleton.
  const { givers, recipients, prevGiverRanks, prevRecipientRanks, names, avatars } = data;

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
    <div
      className="grid grid-cols-1 md:grid-cols-2 gap-8 p-2 transition-opacity"
      style={{ opacity: isFetching ? 0.6 : 1 }}
      aria-busy={isFetching}
    >
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
