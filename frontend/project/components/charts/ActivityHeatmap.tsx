"use client";

import { useEffect, useState, useMemo, useCallback, useRef } from "react";
import { ChartContainer } from "./ChartContainer";
import { defaultChartColors } from "@/lib/chartTheme";
import { buildHeatmapModel, type HeatmapPoint } from "./heatmapModel";
import { HeatmapGrid, CELL_SIZE } from "./HeatmapGrid";
import { HeatmapTooltip, type HeatmapTooltipHandle } from "./HeatmapTooltip";

interface ActivityHeatmapProps {
  startDate: string;
  endDate: string;
  /** `null` is what the backend sends for an empty range: nothing to fetch, nothing to show. */
  preloadedData?: Array<{ date: string; count: number }> | null;
}

export function ActivityHeatmap({ startDate, endDate, preloadedData }: ActivityHeatmapProps) {
  const [data, setData] = useState<HeatmapPoint[]>([]);
  const [loading, setLoading] = useState(preloadedData === undefined);
  const [error, setError] = useState<string | null>(null);

  // The tooltip lives in its own component and is driven through this ref, so hovering a
  // cell never re-renders this component or the grid.
  const tooltipRef = useRef<HeatmapTooltipHandle>(null);
  const handleCellEnter = useCallback((el: HTMLElement, date: string, count: number) => {
    tooltipRef.current?.show(el, date, count);
  }, []);
  const handleCellLeave = useCallback(() => {
    tooltipRef.current?.hide();
  }, []);

  useEffect(() => {
    if (preloadedData !== undefined) {
      setData(preloadedData ?? []);
      setLoading(false);
      return;
    }

    // Ignore the response of a superseded request (range changed or unmounted meanwhile).
    const controller = new AbortController();

    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ start: startDate, end: endDate });
        const resp = await fetch(`/api/proxy/stats/heatmap?${params}`, { signal: controller.signal });
        if (!resp.ok) throw new Error("Failed to fetch heatmap data");
        const json = await resp.json();
        if (controller.signal.aborted) return;
        setData(json || []);
      } catch (err) {
        if (controller.signal.aborted) return;
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }
    fetchData();

    return () => controller.abort();
  }, [startDate, endDate, preloadedData]);

  const colors = defaultChartColors;

  // Build heatmap grid (O(days); cells are looked up by week and weekday)
  const { cells, weeks } = useMemo(
    () => buildHeatmapModel(data, startDate, endDate),
    [data, startDate, endDate]
  );

  return (
    <ChartContainer
      title="Activity Heatmap"
      subtitle="Daily beer activity"
      loading={loading}
      error={error}
    >
      <div className="relative">
        <HeatmapTooltip ref={tooltipRef} />

        <div className="overflow-x-auto">
          <HeatmapGrid
            weeks={weeks}
            cells={cells}
            palette={colors.heatmap}
            onCellEnter={handleCellEnter}
            onCellLeave={handleCellLeave}
          />

          {/* Legend */}
          <div className="flex items-center gap-2 mt-4 text-xs" style={{ color: "hsl(var(--foreground))" }}>
            <span>Less</span>
            {colors.heatmap.map((color, i) => (
              <div
                key={i}
                style={{
                  width: CELL_SIZE,
                  height: CELL_SIZE,
                  backgroundColor: color,
                  borderRadius: 2,
                }}
              />
            ))}
            <span>More</span>
          </div>
        </div>
      </div>
    </ChartContainer>
  );
}
