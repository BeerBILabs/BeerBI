"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import DateRangePicker from "@/components/DateRangePicker";
import { formatLocalDate } from "@/lib/dateUtils";
import { useAnalyticsData } from "@/lib/useAnalyticsData";
import { ChartSkeleton } from "@/components/charts/ChartContainer";
import { useHydrated } from "@/lib/useHydrated";

// Recharts is heavy (~400 KB); load the charts on demand so no other page pays for it.
// Each chart is client-only, so there is nothing to server-render. The placeholders repeat the
// title, subtitle and height of the real chart so the layout does not jump when it appears.
const loadActivityChart = () => import("@/components/charts/ActivityChart").then((m) => m.ActivityChart);
const loadLeaderboardChart = () => import("@/components/charts/LeaderboardChart").then((m) => m.LeaderboardChart);
const loadQuarterlyChart = () => import("@/components/charts/QuarterlyChart").then((m) => m.QuarterlyChart);
const loadActivityHeatmap = () => import("@/components/charts/ActivityHeatmap").then((m) => m.ActivityHeatmap);
const loadNetworkChart = () => import("@/components/charts/NetworkChart").then((m) => m.NetworkChart);

const ActivityChart = dynamic(loadActivityChart, {
  ssr: false,
  loading: () => (
    <ChartSkeleton title="Activity Over Time" subtitle="Beer giving and receiving trends" skeletonHeight="h-72" />
  ),
});
// Two entries for the same module so each placeholder carries the right heading.
const TopGiversChart = dynamic(loadLeaderboardChart, {
  ssr: false,
  loading: () => <ChartSkeleton title="Top Givers" subtitle="Top 10 by beer count" skeletonHeight="h-64" />,
});
const TopRecipientsChart = dynamic(loadLeaderboardChart, {
  ssr: false,
  loading: () => <ChartSkeleton title="Top Recipients" subtitle="Top 10 by beer count" skeletonHeight="h-64" />,
});
const QuarterlyChart = dynamic(loadQuarterlyChart, {
  ssr: false,
  loading: () => {
    const year = new Date().getFullYear();
    return <ChartSkeleton title="Quarterly Comparison" subtitle={`${year - 1} - ${year}`} skeletonHeight="h-64" />;
  },
});
const ActivityHeatmap = dynamic(loadActivityHeatmap, {
  ssr: false,
  loading: () => <ChartSkeleton title="Activity Heatmap" subtitle="Daily beer activity" skeletonHeight="h-36" />,
});
const NetworkChart = dynamic(loadNetworkChart, {
  ssr: false,
  loading: () => (
    <ChartSkeleton title="🤝 Who Appreciates Whom" subtitle="Top giver → recipient pairs" skeletonHeight="h-80" />
  ),
});

type Range = { start: string; end: string };

/** Default range: the current year up to today. */
function thisYearSoFar(): Range {
  const now = new Date();
  return { start: formatLocalDate(new Date(now.getFullYear(), 0, 1)), end: formatLocalDate(now) };
}

export default function AnalyticsPage() {
  // The default range comes from the client's clock after hydration; the page HTML is built once
  // at build time and must not contain a date (see useHydrated).
  const hydrated = useHydrated();
  const [picked, setRange] = useState<Range | null>(null);
  const range = picked ?? (hydrated ? thisYearSoFar() : null);

  // Start loading the chart code right away, in parallel with the data request, so the chunks
  // are already there when the data arrives (otherwise: data first, then JS).
  useEffect(() => {
    void Promise.all([
      loadActivityChart(),
      loadLeaderboardChart(),
      loadQuarterlyChart(),
      loadActivityHeatmap(),
      loadNetworkChart(),
    ]);
  }, []);

  // Fetch all analytics data with a single request. While a new range loads, the previous data
  // stays on screen (charts stay mounted) and `isFetching` drives an inline indicator.
  const { data: analyticsData, dataRange, isFetching, loading, error } = useAnalyticsData({
    startDate: range?.start ?? "",
    endDate: range?.end ?? "",
    granularity: "day",
    limit: 10,
    pairsLimit: 15,
    enabled: range !== null,
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
                start={range?.start ?? null}
                end={range?.end ?? null}
                onChange={(s, e) => setRange({ start: s, end: e })}
              />
            </div>
          </div>
        </div>

        {/* Charts Grid */}
        <div className="space-y-6" aria-busy={isFetching}>
          {isFetching && analyticsData && (
            <p
              role="status"
              className="text-sm text-center animate-pulse"
              style={{ color: "hsl(var(--muted-foreground))" }}
            >
              Updating…
            </p>
          )}

          {(loading || !range) && (
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

          {analyticsData && dataRange && (
            <div
              className="space-y-6 transition-opacity"
              style={{ opacity: isFetching ? 0.6 : 1 }}
            >
              {/* Activity Timeline - Full Width */}
              <ActivityChart
                startDate={dataRange.start}
                endDate={dataRange.end}
                preloadedData={analyticsData.timeline}
              />

              {/* Top Givers & Recipients - Side by Side */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <TopGiversChart
                  startDate={dataRange.start}
                  endDate={dataRange.end}
                  type="givers"
                  limit={10}
                  preloadedData={analyticsData.top_givers}
                />
                <TopRecipientsChart
                  startDate={dataRange.start}
                  endDate={dataRange.end}
                  type="recipients"
                  limit={10}
                  preloadedData={analyticsData.top_recipients}
                />
              </div>

              {/* Heatmap & Network Chart - Side by Side */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <ActivityHeatmap
                  startDate={dataRange.start}
                  endDate={dataRange.end}
                  preloadedData={analyticsData.heatmap}
                />
                <NetworkChart
                  startDate={dataRange.start}
                  endDate={dataRange.end}
                  limit={15}
                  preloadedData={analyticsData.pairs}
                />
              </div>

              {/* Quarterly Comparison - Full Width */}
              <QuarterlyChart />
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
