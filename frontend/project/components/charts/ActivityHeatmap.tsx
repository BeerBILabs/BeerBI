"use client";

import { useEffect, useState, useMemo } from "react";
import { ChartContainer } from "./ChartContainer";
import { defaultChartColors } from "@/lib/chartTheme";

interface HeatmapPoint {
  date: string;
  count: number;
}

interface ActivityHeatmapProps {
  startDate: string;
  endDate: string;
}

// Get day of week (0 = Sunday, 6 = Saturday)
function getDayOfWeek(dateStr: string): number {
  return new Date(dateStr).getDay();
}

// Get week number within the year
function getWeekNumber(dateStr: string): number {
  const date = new Date(dateStr);
  const startOfYear = new Date(date.getFullYear(), 0, 1);
  const days = Math.floor((date.getTime() - startOfYear.getTime()) / (24 * 60 * 60 * 1000));
  return Math.floor((days + startOfYear.getDay()) / 7);
}

// Generate all dates between start and end
function generateDateRange(start: string, end: string): string[] {
  const dates: string[] = [];
  const current = new Date(start);
  const endDate = new Date(end);
  
  while (current <= endDate) {
    dates.push(current.toISOString().split("T")[0]);
    current.setDate(current.getDate() + 1);
  }
  return dates;
}

export function ActivityHeatmap({ startDate, endDate }: ActivityHeatmapProps) {
  const [data, setData] = useState<HeatmapPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ start: startDate, end: endDate });
        const resp = await fetch(`/api/proxy/stats/heatmap?${params}`);
        if (!resp.ok) throw new Error("Failed to fetch heatmap data");
        const json = await resp.json();
        setData(json || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [startDate, endDate]);

  const colors = defaultChartColors;

  // Build heatmap grid
  const { grid, maxCount, weeks } = useMemo(() => {
    const countMap = new Map<string, number>();
    let max = 0;
    data.forEach((d) => {
      countMap.set(d.date, d.count);
      if (d.count > max) max = d.count;
    });

    const allDates = generateDateRange(startDate, endDate);
    const weekSet = new Set<number>();
    
    const gridData = allDates.map((date) => {
      const week = getWeekNumber(date);
      weekSet.add(week);
      return {
        date,
        count: countMap.get(date) || 0,
        dayOfWeek: getDayOfWeek(date),
        week,
      };
    });

    return { grid: gridData, maxCount: max, weeks: Array.from(weekSet).sort((a, b) => a - b) };
  }, [data, startDate, endDate]);

  // Get color intensity based on count
  function getColor(count: number): string {
    if (count === 0) return colors.heatmap[0];
    if (maxCount === 0) return colors.heatmap[0];
    const intensity = Math.min(Math.ceil((count / maxCount) * 4), 4);
    return colors.heatmap[intensity];
  }

  const dayLabels = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const cellSize = 14;
  const cellGap = 3;

  return (
    <ChartContainer
      title="Activity Heatmap"
      subtitle="Daily beer activity"
      loading={loading}
      error={error}
    >
      <div className="overflow-x-auto">
        <div className="inline-flex gap-1">
          {/* Day labels */}
          <div className="flex flex-col gap-[3px] mr-1 text-xs" style={{ color: "hsl(var(--foreground))" }}>
            {dayLabels.map((day, i) => (
              <div
                key={day}
                style={{ height: cellSize, lineHeight: `${cellSize}px` }}
                className={i % 2 === 0 ? "opacity-70" : "opacity-0"}
              >
                {day}
              </div>
            ))}
          </div>

          {/* Heatmap grid */}
          <div className="flex gap-[3px]">
            {weeks.map((week) => (
              <div key={week} className="flex flex-col gap-[3px]">
                {[0, 1, 2, 3, 4, 5, 6].map((dayOfWeek) => {
                  const cell = grid.find((g) => g.week === week && g.dayOfWeek === dayOfWeek);
                  if (!cell) {
                    return (
                      <div
                        key={dayOfWeek}
                        style={{
                          width: cellSize,
                          height: cellSize,
                          backgroundColor: "transparent",
                        }}
                      />
                    );
                  }
                  return (
                    <div
                      key={dayOfWeek}
                      title={`${new Date(cell.date).toLocaleDateString("de-DE")}: ${cell.count} beers`}
                      style={{
                        width: cellSize,
                        height: cellSize,
                        backgroundColor: getColor(cell.count),
                        borderRadius: 2,
                      }}
                      className="cursor-pointer hover:ring-2 hover:ring-offset-1 transition-all"
                    />
                  );
                })}
              </div>
            ))}
          </div>
        </div>

        {/* Legend */}
        <div className="flex items-center gap-2 mt-4 text-xs" style={{ color: "hsl(var(--foreground))" }}>
          <span>Less</span>
          {colors.heatmap.map((color, i) => (
            <div
              key={i}
              style={{
                width: cellSize,
                height: cellSize,
                backgroundColor: color,
                borderRadius: 2,
              }}
            />
          ))}
          <span>More</span>
        </div>
      </div>
    </ChartContainer>
  );
}
