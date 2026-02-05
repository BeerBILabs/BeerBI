"use client";

import { useState } from "react";
import DateRangePicker from "@/components/DateRangePicker";
import { formatLocalDate } from "@/lib/dateUtils";
import { useAnalyticsData } from "@/lib/useAnalyticsData";
import {
  ActivityChart,
  LeaderboardChart,
  QuarterlyChart,
  ActivityHeatmap,
  NetworkChart,
} from "@/components/charts";

export default function AnalyticsPage() {
  const now = new Date();
  const yearStart = new Date(now.getFullYear(), 0, 1);

  const [range, setRange] = useState({
    start: formatLocalDate(yearStart),
    end: formatLocalDate(now),
  });

  // Fetch all analytics data with a single request
  const { data: analyticsData, loading, error } = useAnalyticsData({
    startDate: range.start,
    endDate: range.end,
    granularity: "day",
    limit: 10,
    pairsLimit: 15,
  });

  const quickRanges = [
    {
      label: "This Year",
      get: () => {
        const today = new Date();
        return {
          start: formatLocalDate(new Date(today.getFullYear(), 0, 1)),
          end: formatLocalDate(today),
        };
      },
    },
    {
      label: "Last Year",
      get: () => {
        const today = new Date();
        return {
          start: formatLocalDate(new Date(today.getFullYear() - 1, 0, 1)),
          end: formatLocalDate(new Date(today.getFullYear() - 1, 11, 31)),
        };
      },
    },
    {
      label: "Last 6 Months",
      get: () => {
        const today = new Date();
        const d = new Date();
        d.setMonth(d.getMonth() - 6);
        return { start: formatLocalDate(d), end: formatLocalDate(today) };
      },
    },
    {
      label: "Last 3 Months",
      get: () => {
        const today = new Date();
        const d = new Date();
        d.setMonth(d.getMonth() - 3);
        return { start: formatLocalDate(d), end: formatLocalDate(today) };
      },
    },
  ];

  return (
    <main className="min-h-screen p-6 md:p-10">
      <div className="w-full max-w-full px-0 md:px-4">
        {/* Header */}
        <div className="mb-8">
          <h1
            className="text-3xl font-extrabold tracking-tight"
            style={{ color: "hsl(var(--primary))" }}
          >
            BeerBI Analytics
          </h1>
          <p
            className="mt-2 text-base"
            style={{ color: "hsl(var(--muted-foreground))" }}
          >
            Insights and trends from your team's beer-giving activity
          </p>
        </div>

        {/* Date Range Controls */}
        <div
          className="mb-8 p-4 rounded-xl border"
          style={{
            backgroundColor: "hsl(var(--card))",
            borderColor: "hsl(var(--border))",
          }}
        >
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex flex-wrap gap-2">
              {quickRanges.map((q) => (
                <button
                  key={q.label}
                  type="button"
                  className="px-3 py-1.5 rounded text-sm font-medium border transition-colors"
                  style={{
                    backgroundColor: "hsl(var(--secondary))",
                    color: "hsl(var(--secondary-foreground))",
                    borderColor: "hsl(var(--border))",
                  }}
                  onClick={() => setRange(q.get())}
                >
                  {q.label}
                </button>
              ))}
            </div>
            <div className="flex-1 flex justify-end">
              <DateRangePicker
                start={range.start}
                end={range.end}
                onChange={(s, e) => setRange({ start: s, end: e })}
              />
            </div>
          </div>
        </div>

        {/* Charts Grid */}
        <div className="space-y-6">
          {loading && (
            <div className="text-center py-12">
              <p style={{ color: "hsl(var(--muted-foreground))" }}>
                Loading analytics...
              </p>
            </div>
          )}

          {error && (
            <div
              className="p-4 rounded-lg border"
              style={{
                backgroundColor: "hsl(var(--destructive) / 0.1)",
                borderColor: "hsl(var(--destructive))",
                color: "hsl(var(--destructive))",
              }}
            >
              Error loading analytics: {error}
            </div>
          )}

          {analyticsData && (
            <>
              {/* Activity Timeline - Full Width */}
              <ActivityChart
                startDate={range.start}
                endDate={range.end}
                preloadedData={analyticsData.timeline}
              />

              {/* Top Givers & Recipients - Side by Side */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <LeaderboardChart
                  startDate={range.start}
                  endDate={range.end}
                  type="givers"
                  limit={10}
                  preloadedData={analyticsData.top_givers}
                />
                <LeaderboardChart
                  startDate={range.start}
                  endDate={range.end}
                  type="recipients"
                  limit={10}
                  preloadedData={analyticsData.top_recipients}
                />
              </div>

              {/* Heatmap & Network Chart - Side by Side */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <ActivityHeatmap
                  startDate={range.start}
                  endDate={range.end}
                  preloadedData={analyticsData.heatmap}
                />
                <NetworkChart
                  startDate={range.start}
                  endDate={range.end}
                  limit={15}
                  preloadedData={analyticsData.pairs}
                />
              </div>

              {/* Quarterly Comparison - Full Width */}
              <QuarterlyChart />
            </>
          )}
        </div>
      </div>
    </main>
  );
}
