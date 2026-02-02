"use client";

import { useState } from "react";
import Link from "next/link";
import DateRangePicker from "@/components/DateRangePicker";
import {
  ActivityChart,
  LeaderboardChart,
  QuarterlyChart,
  ActivityHeatmap,
  NetworkChart,
} from "@/components/charts";

function formatLocalDate(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export default function AnalyticsPage() {
  const now = new Date();
  const yearStart = new Date(now.getFullYear(), 0, 1);

  const [range, setRange] = useState({
    start: formatLocalDate(yearStart),
    end: formatLocalDate(now),
  });

  const quickRanges = [
    {
      label: "This Year",
      get: () => ({
        start: formatLocalDate(new Date(now.getFullYear(), 0, 1)),
        end: formatLocalDate(now),
      }),
    },
    {
      label: "Last Year",
      get: () => ({
        start: formatLocalDate(new Date(now.getFullYear() - 1, 0, 1)),
        end: formatLocalDate(new Date(now.getFullYear() - 1, 11, 31)),
      }),
    },
    {
      label: "Last 6 Months",
      get: () => {
        const d = new Date();
        d.setMonth(d.getMonth() - 6);
        return { start: formatLocalDate(d), end: formatLocalDate(now) };
      },
    },
    {
      label: "Last 3 Months",
      get: () => {
        const d = new Date();
        d.setMonth(d.getMonth() - 3);
        return { start: formatLocalDate(d), end: formatLocalDate(now) };
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
          {/* Activity Timeline - Full Width */}
          <ActivityChart startDate={range.start} endDate={range.end} />

          {/* Top Givers & Recipients - Side by Side */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <LeaderboardChart
              startDate={range.start}
              endDate={range.end}
              type="givers"
              limit={10}
            />
            <LeaderboardChart
              startDate={range.start}
              endDate={range.end}
              type="recipients"
              limit={10}
            />
          </div>

          {/* Heatmap & Network Chart - Side by Side */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <ActivityHeatmap startDate={range.start} endDate={range.end} />
            <NetworkChart startDate={range.start} endDate={range.end} limit={15} />
          </div>

          {/* Quarterly Comparison - Full Width */}
          <QuarterlyChart />
        </div>

        {/* Footer */}
        <div className="mt-8 text-center">
          <Link
            href="/rankings"
            className="px-4 py-2 rounded-md text-sm font-semibold transition-colors inline-block"
            style={{
              backgroundColor: "hsl(var(--primary))",
              color: "hsl(var(--primary-foreground))",
            }}
          >
            View Quarterly Rankings
          </Link>
        </div>
      </div>
    </main>
  );
}
