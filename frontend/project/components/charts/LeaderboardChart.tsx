"use client";

import { useEffect, useState } from "react";
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

interface TopUserStats {
  userId: string;
  count: number;
}

interface TopUsersResult {
  givers: TopUserStats[];
  recipients: TopUserStats[];
}

interface LeaderboardChartProps {
  startDate: string;
  endDate: string;
  type: "givers" | "recipients";
  limit?: number;
}

export function LeaderboardChart({
  startDate,
  endDate,
  type,
  limit = 10,
}: LeaderboardChartProps) {
  const [data, setData] = useState<TopUserStats[]>([]);
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
        const resp = await fetch(`/api/proxy/stats/top?${params}`);
        if (!resp.ok) throw new Error("Failed to fetch top users");
        const json: TopUsersResult = await resp.json();
        const users = type === "givers" ? json.givers : json.recipients;
        setData(users || []);

        // Fetch names for users
        const nameMap: Record<string, string> = {};
        await Promise.all(
          (users || []).slice(0, 10).map(async (u) => {
            try {
              const userResp = await fetch(`/api/proxy/user?user=${encodeURIComponent(u.userId)}`);
              if (userResp.ok) {
                const userData = await userResp.json();
                nameMap[u.userId] = userData.real_name || u.userId;
              } else {
                nameMap[u.userId] = u.userId;
              }
            } catch {
              nameMap[u.userId] = u.userId;
            }
          })
        );
        setNames(nameMap);
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [startDate, endDate, type, limit]);

  const colors = defaultChartColors;
  const barColor = type === "givers" ? colors.given : colors.received;
  const title = type === "givers" ? "Top Givers" : "Top Recipients";

  // Prepare chart data with names
  const chartData = data.map((d) => ({
    ...d,
    name: names[d.userId] || d.userId,
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
              formatter={(value: number) => [`${value}`, "Beers"]}
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
