import { describe, expect, it } from 'vitest';
import { buildHeatmapModel, cellKey, getIntensity } from '../charts/heatmapModel';
import { formatLocalDate, parseLocalDate } from '@/lib/dateUtils';

describe('getIntensity', () => {
  it('returns 0 for empty days and when nothing was counted', () => {
    expect(getIntensity(0, 10)).toBe(0);
    expect(getIntensity(5, 0)).toBe(0);
  });

  it('buckets counts relative to the busiest day into levels 1-4', () => {
    expect(getIntensity(1, 100)).toBe(1);
    expect(getIntensity(25, 100)).toBe(1);
    expect(getIntensity(26, 100)).toBe(2);
    expect(getIntensity(51, 100)).toBe(3);
    expect(getIntensity(76, 100)).toBe(4);
    expect(getIntensity(100, 100)).toBe(4);
  });
});

describe('buildHeatmapModel', () => {
  it('creates one cell per day, indexed by week and weekday', () => {
    // 2025-01-01 is a Wednesday, 2025-01-31 a Friday
    const model = buildHeatmapModel([], '2025-01-01', '2025-01-31');
    expect(model.cells.size).toBe(31);
    for (const cell of model.cells.values()) {
      expect(model.cells.get(cellKey(cell.weekKey, cell.dayOfWeek))).toBe(cell);
      expect(parseLocalDate(cell.date).getDay()).toBe(cell.dayOfWeek);
    }
    // Sunday-based weeks: Jan 1-4, 5-11, 12-18, 19-25, 26-31
    expect(model.weeks).toHaveLength(5);
    const firstWeek = model.weeks[0];
    expect(model.cells.has(cellKey(firstWeek, 0))).toBe(false); // Sun Dec 29 is before the range
    expect(model.cells.has(cellKey(firstWeek, 3))).toBe(true); // Wed Jan 1
    const lastWeek = model.weeks[model.weeks.length - 1];
    expect(model.cells.has(cellKey(lastWeek, 5))).toBe(true); // Fri Jan 31
    expect(model.cells.has(cellKey(lastWeek, 6))).toBe(false); // Sat Feb 1 is after the range
  });

  it('puts the right count into the right cell and defaults to zero', () => {
    const model = buildHeatmapModel(
      [
        { date: '2025-01-02', count: 3 },
        { date: '2025-01-15', count: 12 },
      ],
      '2025-01-01',
      '2025-01-31'
    );
    const byDate = new Map([...model.cells.values()].map((c) => [c.date, c]));
    expect(byDate.get('2025-01-02')?.count).toBe(3);
    expect(byDate.get('2025-01-15')?.count).toBe(12);
    expect(byDate.get('2025-01-15')?.intensity).toBe(4);
    expect(byDate.get('2025-01-02')?.intensity).toBe(1);
    expect(byDate.get('2025-01-03')?.count).toBe(0);
    expect(byDate.get('2025-01-03')?.intensity).toBe(0);
    expect(model.maxCount).toBe(12);
  });

  it('keeps the week columns in calendar order', () => {
    const model = buildHeatmapModel([], '2025-01-01', '2025-04-30');
    const firstDates = model.weeks.map((w) =>
      [0, 1, 2, 3, 4, 5, 6].map((d) => model.cells.get(cellKey(w, d))?.date).find(Boolean) as string
    );
    expect(firstDates).toEqual([...firstDates].sort());
    expect(new Set(model.weeks).size).toBe(model.weeks.length);
  });

  it('keeps a week that spans New Year in one column', () => {
    // Sun Dec 29 2024 to Sat Jan 4 2025 is one calendar week
    const model = buildHeatmapModel([], '2024-12-20', '2025-01-10');
    expect(model.weeks).toEqual(['2024-12-15', '2024-12-22', '2024-12-29', '2025-01-05']);
    expect(model.cells.size).toBe(22);
    const column = [0, 1, 2, 3, 4, 5, 6].map((d) => model.cells.get(cellKey('2024-12-29', d))?.date);
    expect(column).toEqual([
      '2024-12-29',
      '2024-12-30',
      '2024-12-31',
      '2025-01-01',
      '2025-01-02',
      '2025-01-03',
      '2025-01-04',
    ]);
  });

  it('keeps every day in its own cell across the DST changes (TZ=Europe/Berlin)', () => {
    // 2025: DST starts Mar 30 and ends Oct 26. Every Sunday must get its own column.
    const model = buildHeatmapModel([], '2025-01-01', '2025-12-31');
    expect(model.cells.size).toBe(365);
    const sundays = [...model.cells.values()].filter((c) => c.dayOfWeek === 0);
    expect(sundays).toHaveLength(52);
    expect(new Set(sundays.map((c) => c.weekKey)).size).toBe(52);
    // The columns of a summer week hold Sunday to Saturday of the same calendar week.
    expect(model.cells.get(cellKey('2025-06-29', 0))?.date).toBe('2025-06-29');
    expect(model.cells.get(cellKey('2025-06-29', 6))?.date).toBe('2025-07-05');
    expect(model.cells.get(cellKey('2025-10-26', 0))?.date).toBe('2025-10-26'); // DST ends
  });

  it('builds a 2000-day grid quickly (linear time)', () => {
    const start = '2020-01-01';
    const startDate = parseLocalDate(start);
    const end = new Date(startDate);
    end.setDate(end.getDate() + 1999);
    const data = Array.from({ length: 2000 }, (_, i) => {
      const d = new Date(startDate);
      d.setDate(d.getDate() + i);
      return { date: formatLocalDate(d), count: (i * 7) % 23 };
    });

    buildHeatmapModel(data, start, formatLocalDate(end)); // warm-up: JIT and first-call costs

    const t0 = performance.now();
    const model = buildHeatmapModel(data, start, formatLocalDate(end));
    const elapsed = performance.now() - t0;

    expect(model.cells.size).toBe(2000);
    // Typically a few ms; the limit leaves room for slow CI runners while still catching O(days^2).
    expect(elapsed).toBeLessThan(200);
  });
});
