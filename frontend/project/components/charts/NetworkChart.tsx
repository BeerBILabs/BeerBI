"use client";

import { useEffect, useState } from "react";
import { ChartContainer } from "./ChartContainer";
import { defaultChartColors } from "@/lib/chartTheme";
import { userDataManager } from "@/lib/userCache";

interface PairStats {
  giver: string;
  recipient: string;
  count: number;
}

interface NetworkChartProps {
  startDate: string;
  endDate: string;
  limit?: number;
}

export function NetworkChart({ startDate, endDate, limit = 20 }: NetworkChartProps) {
  const [data, setData] = useState<PairStats[]>([]);
  const [names, setNames] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({
          start: startDate,
          end: endDate,
          limit: limit.toString(),
        });
        const resp = await fetch(`/api/proxy/stats/pairs?${params}`);
        if (!resp.ok) throw new Error("Failed to fetch pair data");
        const json = await resp.json();
        setData(json || []);

        // Collect all unique user IDs
        const userIds = new Set<string>();
        (json || []).forEach((p: PairStats) => {
          userIds.add(p.giver);
          userIds.add(p.recipient);
        });

        // Batch fetch user names using UserDataManager
        if (userIds.size > 0) {
          const userInfos = await userDataManager.getUsers(Array.from(userIds));
          const nameMap: Record<string, string> = {};
          for (const [userId, info] of Object.entries(userInfos)) {
            nameMap[userId] = info.real_name;
          }
          // Fill in any missing users with userId as fallback
          for (const userId of userIds) {
            if (!nameMap[userId]) {
              nameMap[userId] = userId;
            }
          }
          setNames(nameMap);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [startDate, endDate, limit]);

  const colors = defaultChartColors;

  // Calculate max count for scaling
  const maxCount = Math.max(...data.map((d) => d.count), 1);

  if (!loading && data.length === 0) {
    return (
      <ChartContainer
        title="Who Appreciates Whom"
        subtitle="Giver → Recipient relationships"
        loading={false}
        error={null}
      >
        <div className="h-64 flex items-center justify-center">
          <p style={{ color: colors.foreground }} className="text-sm opacity-70">
            No pair data available for this period
          </p>
        </div>
      </ChartContainer>
    );
  }

  return (
    <ChartContainer
      title="🤝 Who Appreciates Whom"
      subtitle="Top giver → recipient pairs"
      loading={loading}
      error={error}
    >
      <div className="space-y-2 max-h-80 overflow-y-auto pr-2">
        {data.map((pair, index) => {
          const giverName = names[pair.giver] || pair.giver;
          const recipientName = names[pair.recipient] || pair.recipient;
          const widthPercent = (pair.count / maxCount) * 100;

          return (
            <div key={`${pair.giver}-${pair.recipient}`} className="flex items-center gap-2">
              <div
                className="text-sm font-medium w-30 truncate text-right"
                style={{ color: "hsl(var(--foreground))" }}
                title={giverName}
              >
                {giverName}
              </div>
              <div className="w-40 relative h-5">
                <div
                  className="absolute inset-y-0 left-0 rounded-r flex items-center justify-end pr-1.5"
                  style={{
                    width: `${Math.max(widthPercent, 50)}%`,
                    backgroundColor: colors.given,
                    opacity: 0.9 - index * 0.03,
                  }}
                >
                  <span className="text-xs font-bold" style={{ color: "hsl(var(--primary-foreground))" }}>
                    {pair.count}
                  </span>
                </div>
              </div>
              <div
                className="text-sm font-medium w-30 truncate text-left"
                style={{ color: "hsl(var(--foreground))" }}
                title={recipientName}
              >
                {recipientName}
              </div>
            </div>
          );
        })}
      </div>
    </ChartContainer>
  );
}
