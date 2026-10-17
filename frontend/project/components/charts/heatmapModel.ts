import { formatLocalDate, generateDateRange, parseLocalDate } from "@/lib/dateUtils";

export interface HeatmapPoint {
  date: string;
  count: number;
}

export type HeatmapIntensity = 0 | 1 | 2 | 3 | 4;

export interface HeatmapCell {
  date: string;
  count: number;
  dayOfWeek: number;
  weekKey: string;
  /** Index into the heatmap palette (0 = empty, 4 = busiest). */
  intensity: HeatmapIntensity;
}

export interface HeatmapModel {
  /** Week columns in chronological order. */
  weeks: string[];
  /** Cells by `cellKey(weekKey, dayOfWeek)` for O(1) lookup. */
  cells: Map<string, HeatmapCell>;
  maxCount: number;
}

export function cellKey(weekKey: string, dayOfWeek: number): string {
  return `${weekKey}:${dayOfWeek}`;
}

// A column is one calendar week (Sunday to Saturday), keyed by the date of its Sunday.
// Local-date arithmetic keeps this correct across DST changes and year boundaries, so a week
// that spans New Year stays in one column.
function getWeekKey(date: Date): string {
  return formatLocalDate(new Date(date.getFullYear(), date.getMonth(), date.getDate() - date.getDay()));
}

/** Colour bucket for a count: 0 for empty, otherwise 1-4 relative to the busiest day. */
export function getIntensity(count: number, maxCount: number): HeatmapIntensity {
  if (count <= 0 || maxCount <= 0) return 0;
  return Math.min(Math.ceil((count / maxCount) * 4), 4) as HeatmapIntensity;
}

/**
 * Builds the calendar grid for every day in [startDate, endDate].
 * Runs in O(days): the cells are indexed by week and weekday instead of being searched per render.
 */
export function buildHeatmapModel(data: HeatmapPoint[], startDate: string, endDate: string): HeatmapModel {
  const countMap = new Map<string, number>();
  let maxCount = 0;
  for (const d of data) {
    countMap.set(d.date, d.count);
    if (d.count > maxCount) maxCount = d.count;
  }

  // Dates are generated chronologically, so the Set keeps the weeks in calendar order.
  // (Sorting the keys as strings would put "2025-W10" before "2025-W2".)
  const weekSet = new Set<string>();
  const cells = new Map<string, HeatmapCell>();
  for (const date of generateDateRange(startDate, endDate)) {
    const parsed = parseLocalDate(date);
    const weekKey = getWeekKey(parsed);
    const dayOfWeek = parsed.getDay();
    const count = countMap.get(date) || 0;
    weekSet.add(weekKey);
    cells.set(cellKey(weekKey, dayOfWeek), {
      date,
      count,
      dayOfWeek,
      weekKey,
      intensity: getIntensity(count, maxCount),
    });
  }

  return { weeks: Array.from(weekSet), cells, maxCount };
}
