"use client";

import { memo } from "react";
import { cellKey, type HeatmapModel } from "./heatmapModel";

const DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const DAYS_OF_WEEK = [0, 1, 2, 3, 4, 5, 6];
export const CELL_SIZE = 14;
export const CELL_GAP = 3;

interface HeatmapGridProps {
  weeks: HeatmapModel["weeks"];
  cells: HeatmapModel["cells"];
  /** One colour per intensity level, index 0 = empty. */
  palette: string[];
  onCellEnter: (el: HTMLElement, date: string, count: number) => void;
  onCellLeave: () => void;
}

/**
 * The calendar grid. Memoized: it only re-renders when the data or the (stable) callbacks change,
 * never because the tooltip moved.
 */
export const HeatmapGrid = memo(function HeatmapGrid({
  weeks,
  cells,
  palette,
  onCellEnter,
  onCellLeave,
}: HeatmapGridProps) {
  return (
    <div className="inline-flex gap-1">
      {/* Day labels */}
      <div className="flex flex-col mr-1 text-xs" style={{ color: "hsl(var(--foreground))", gap: CELL_GAP }}>
        {DAY_LABELS.map((day, i) => (
          <div
            key={day}
            style={{ height: CELL_SIZE, lineHeight: `${CELL_SIZE}px` }}
            className={i % 2 === 0 ? "opacity-70" : "opacity-0"}
          >
            {day}
          </div>
        ))}
      </div>

      {/* Heatmap grid */}
      <div className="flex" style={{ gap: CELL_GAP }}>
        {weeks.map((weekKey) => (
          <div key={weekKey} className="flex flex-col" style={{ gap: CELL_GAP }}>
            {DAYS_OF_WEEK.map((dayOfWeek) => {
              const cell = cells.get(cellKey(weekKey, dayOfWeek));
              if (!cell) {
                return (
                  <div
                    key={dayOfWeek}
                    data-testid="heatmap-empty"
                    style={{
                      width: CELL_SIZE,
                      height: CELL_SIZE,
                      backgroundColor: "transparent",
                    }}
                  />
                );
              }
              return (
                <div
                  key={dayOfWeek}
                  data-testid="heatmap-cell"
                  data-date={cell.date}
                  data-count={cell.count}
                  onMouseEnter={(e) => onCellEnter(e.currentTarget, cell.date, cell.count)}
                  onMouseLeave={onCellLeave}
                  style={{
                    width: CELL_SIZE,
                    height: CELL_SIZE,
                    backgroundColor: palette[cell.intensity],
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
  );
});
