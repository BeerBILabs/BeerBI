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
import { userDataManager } from "@/lib/userCache";

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
  preloadedData?: Array<{ user_id: string; count: number }>;
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
    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        let users: TopUserStats[];

        if (preloadedData) {
          // Use preloaded data
          users = preloadedData.map((u) => ({
            userId: u.user_id,
            count: u.count,
          }));
        } else {
          // Fetch from API
          const params = new URLSearchParams({
            start: startDate,
            end: endDate,
            limit: limit.toString(),
          });
          const resp = await fetch(`/api/proxy/stats/top?${params}`);
          if (!resp.ok) throw new Error("Failed to fetch top users");
          const json: TopUsersResult = await resp.json();
          users = type === "givers" ? json.givers : json.recipients;
        }

        setData(users || []);

        // Batch fetch user names using UserDataManager
        const userIds = (users || []).slice(0, 10).map((u) => u.userId);
        if (userIds.length > 0) {
          const userInfos = await userDataManager.getUsers(userIds);
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
  }, [startDate, endDate, type, limit, preloadedData]);

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
