"use client";

import { useEffect, useMemo, useState } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { ChartContainer } from "./ChartContainer";
import { defaultChartColors, formatNumber } from "@/lib/chartTheme";
import { knownUserNames, resolveUsers, topUserRefs } from "@/lib/userNames";
import type { TopUserStats, TopUsersResult } from "@/lib/statsTypes";

interface LeaderboardChartProps {
  startDate: string;
  endDate: string;
  type: "givers" | "recipients";
  limit?: number;
  /** Rows from /stats/combined; names are taken from them when the backend sends them inline. */
  preloadedData?: TopUserStats[];
}

export function LeaderboardChart({
  startDate,
  endDate,
  type,
  limit = 10,
  preloadedData,
}: LeaderboardChartProps) {
  const [data, setData] = useState<TopUserStats[]>([]);
  const [names, setNames] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(!preloadedData);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    async function fetchData() {
      setError(null);
      try {
        let users: TopUserStats[];

        if (preloadedData) {
          // Preloaded rows are shown at once; the chart stays mounted while names resolve.
          users = preloadedData;
        } else {
          setLoading(true);
          const params = new URLSearchParams({
            start: startDate,
            end: endDate,
            limit: limit.toString(),
          });
          const resp = await fetch(`/api/proxy/stats/top?${params}`, { signal: controller.signal });
          if (!resp.ok) throw new Error("Failed to fetch top users");
          const json: TopUsersResult = await resp.json();
          users = type === "givers" ? json.givers : json.recipients;
        }
        if (controller.signal.aborted) return;
        setData(users || []);

        const shown = (users || []).slice(0, 10);
        if (shown.length > 0) {
          const resolved = await resolveUsers(topUserRefs(shown));
          if (controller.signal.aborted) return;
          setNames(Object.fromEntries(Object.entries(resolved).map(([id, u]) => [id, u.name])));
        }
      } catch (err) {
        if (controller.signal.aborted) return;
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }
    fetchData();

    return () => controller.abort();
  }, [startDate, endDate, type, limit, preloadedData]);

  const colors = defaultChartColors;
  const barColor = type === "givers" ? colors.given : colors.received;
  const title = type === "givers" ? "Top Givers" : "Top Recipients";

  // Names known without a request (inline or cached) label new rows at once; `names` may still
  // belong to the previous rows while resolveUsers runs.
  const quickNames = useMemo(() => knownUserNames(topUserRefs(data)), [data]);
  const chartData = data.map((d) => ({
    ...d,
    name: quickNames[d.userId] || names[d.userId] || d.userId,
  }));

  return (
    <ChartContainer
      title={title}
      subtitle={`Top ${limit} by beer count`}
      loading={loading}
      error={error}
    >
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={chartData}
            layout="vertical"
            margin={{ top: 5, right: 30, left: 80, bottom: 5 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke={colors.border} horizontal={false} />
            <XAxis
              type="number"
              tickFormatter={formatNumber}
              tick={{ fill: colors.foreground, fontSize: 12 }}
              axisLine={{ stroke: colors.border }}
            />
            <YAxis
              type="category"
              dataKey="name"
              tick={{ fill: colors.foreground, fontSize: 12 }}
              axisLine={{ stroke: colors.border }}
              width={75}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "hsl(var(--card))",
                border: "1px solid hsl(var(--border))",
                borderRadius: "8px",
                color: "hsl(var(--card-foreground))",
              }}
              labelStyle={{ color: "hsl(var(--card-foreground))" }}
              itemStyle={{ color: "hsl(var(--card-foreground))" }}
              formatter={(value) => [`${value}`, "Beers"]}
              cursor={{ fill: "hsl(var(--accent))" }}
            />
            <Bar dataKey="count" radius={[0, 4, 4, 0]}>
              {chartData.map((_, index) => (
                <Cell
                  key={`cell-${index}`}
                  fill={barColor}
                  fillOpacity={1 - index * 0.08}
                />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </ChartContainer>
  );
}
