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
} from "recharts";
import { ChartContainer } from "./ChartContainer";
import { defaultChartColors, formatNumber } from "@/lib/chartTheme";

interface QuarterlyStats {
  year: number;
  quarter: number;
  count: number;
}

interface QuarterlyChartProps {
  startYear?: number;
  endYear?: number;
}

export function QuarterlyChart({ startYear, endYear }: QuarterlyChartProps) {
  const [data, setData] = useState<QuarterlyStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const currentYear = new Date().getFullYear();
  const effectiveStartYear = startYear ?? currentYear - 1;
  const effectiveEndYear = endYear ?? currentYear;

  useEffect(() => {
    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({
          start_year: effectiveStartYear.toString(),
          end_year: effectiveEndYear.toString(),
        });
        const resp = await fetch(`/api/proxy/stats/quarterly?${params}`);
        if (!resp.ok) throw new Error("Failed to fetch quarterly data");
        const json = await resp.json();
        setData(json || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [effectiveStartYear, effectiveEndYear]);

  const colors = defaultChartColors;

  // Format data for chart
  const chartData = data.map((d) => ({
    ...d,
    label: `Q${d.quarter} ${d.year}`,
  }));

  return (
    <ChartContainer
      title="Quarterly Comparison"
      subtitle={`${effectiveStartYear} - ${effectiveEndYear}`}
      loading={loading}
      error={error}
    >
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
            <XAxis
              dataKey="label"
              tick={{ fill: colors.foreground, fontSize: 12 }}
              axisLine={{ stroke: colors.border }}
            />
            <YAxis
              tickFormatter={formatNumber}
              tick={{ fill: colors.foreground, fontSize: 12 }}
              axisLine={{ stroke: colors.border }}
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
            <Bar
              dataKey="count"
              name="Beers"
              fill={colors.given}
              radius={[4, 4, 0, 0]}
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </ChartContainer>
  );
}
