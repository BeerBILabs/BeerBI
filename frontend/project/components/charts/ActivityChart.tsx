"use client";

import { useEffect, useState } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { ChartContainer } from "./ChartContainer";
import { GranularityToggle, type Granularity } from "./GranularityToggle";
import { defaultChartColors, formatDate, formatNumber } from "@/lib/chartTheme";

interface TimelinePoint {
  date: string;
  given: number;
  received: number;
}

interface ActivityChartProps {
  startDate: string;
  endDate: string;
}

export function ActivityChart({ startDate, endDate }: ActivityChartProps) {
  const [data, setData] = useState<TimelinePoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [granularity, setGranularity] = useState<Granularity>("day");

  useEffect(() => {
    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({
          start: startDate,
          end: endDate,
          granularity,
        });
        const resp = await fetch(`/api/proxy/stats/timeline?${params}`);
        if (!resp.ok) throw new Error("Failed to fetch timeline data");
        const json = await resp.json();
        setData(json || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [startDate, endDate, granularity]);

  const colors = defaultChartColors;

  return (
    <ChartContainer
      title="Activity Over Time"
      subtitle="Beer giving and receiving trends"
      loading={loading}
      error={error}
      onRetry={() => setGranularity(granularity)}
    >
      <div className="flex justify-end mb-4">
        <GranularityToggle value={granularity} onChange={setGranularity} />
      </div>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="givenGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={colors.given} stopOpacity={0.8} />
                <stop offset="95%" stopColor={colors.given} stopOpacity={0.1} />
              </linearGradient>
              <linearGradient id="receivedGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={colors.received} stopOpacity={0.8} />
                <stop offset="95%" stopColor={colors.received} stopOpacity={0.1} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
            <XAxis
              dataKey="date"
              tickFormatter={(v) => formatDate(v, granularity)}
              tick={{ fill: colors.foreground, fontSize: 12 }}
              axisLine={{ stroke: colors.border }}
              tickLine={{ stroke: colors.border }}
            />
            <YAxis
              tickFormatter={formatNumber}
              tick={{ fill: colors.foreground, fontSize: 12 }}
              axisLine={{ stroke: colors.border }}
              tickLine={{ stroke: colors.border }}
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
              labelFormatter={(v) => formatDate(v as string, granularity)}
              cursor={{ stroke: "hsl(var(--accent))", strokeWidth: 2 }}
            />
            <Legend />
            <Area
              type="monotone"
              dataKey="given"
              name="Given"
              stroke={colors.given}
              fill="url(#givenGradient)"
              strokeWidth={2}
            />
            <Area
              type="monotone"
              dataKey="received"
              name="Received"
              stroke={colors.received}
              fill="url(#receivedGradient)"
              strokeWidth={2}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </ChartContainer>
  );
}
