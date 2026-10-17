import { describe, expect, it } from 'vitest';
import { formatLocalDate, generateDateRange, parseLocalDate } from '../dateUtils';

describe('test environment', () => {
  it('runs in Europe/Berlin so timezone bugs show up on UTC runners', () => {
    expect(process.env.TZ).toBe('Europe/Berlin');
    // Winter: UTC+1, summer: UTC+2
    expect(new Date(2026, 0, 15).getTimezoneOffset()).toBe(-60);
    expect(new Date(2026, 6, 15).getTimezoneOffset()).toBe(-120);
  });
});

describe('formatLocalDate', () => {
  it('formats local midnight without shifting to the previous UTC day', () => {
    expect(formatLocalDate(new Date(2026, 0, 1))).toBe('2026-01-01');
  });

  it('pads month and day', () => {
    expect(formatLocalDate(new Date(2026, 2, 5))).toBe('2026-03-05');
  });

  it('uses local calendar fields late in the evening', () => {
    expect(formatLocalDate(new Date(2026, 11, 31, 23, 59, 59))).toBe('2026-12-31');
  });
});

describe('parseLocalDate', () => {
  it('returns local midnight of the given day', () => {
    const d = parseLocalDate('2026-03-31');
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(2);
    expect(d.getDate()).toBe(31);
    expect(d.getHours()).toBe(0);
  });

  it('round-trips with formatLocalDate', () => {
    for (const s of ['2024-02-29', '2026-01-01', '2026-10-25', '2026-12-31']) {
      expect(formatLocalDate(parseLocalDate(s))).toBe(s);
    }
  });
});

describe('generateDateRange', () => {
  it('is inclusive on both ends', () => {
    expect(generateDateRange('2026-01-01', '2026-01-03')).toEqual([
      '2026-01-01',
      '2026-01-02',
      '2026-01-03',
    ]);
  });

  it('returns one entry for a single day', () => {
    expect(generateDateRange('2026-05-05', '2026-05-05')).toEqual(['2026-05-05']);
  });

  it('returns an empty list when start is after end', () => {
    expect(generateDateRange('2026-05-06', '2026-05-05')).toEqual([]);
  });

  it('crosses month ends', () => {
    expect(generateDateRange('2026-01-30', '2026-02-02')).toEqual([
      '2026-01-30',
      '2026-01-31',
      '2026-02-01',
      '2026-02-02',
    ]);
  });

  it('handles leap day', () => {
    expect(generateDateRange('2024-02-28', '2024-03-01')).toEqual([
      '2024-02-28',
      '2024-02-29',
      '2024-03-01',
    ]);
    expect(generateDateRange('2026-02-28', '2026-03-01')).toHaveLength(2);
  });

  it('crosses the year boundary', () => {
    expect(generateDateRange('2025-12-31', '2026-01-01')).toEqual(['2025-12-31', '2026-01-01']);
  });

  it('has no gaps or duplicates over the spring DST change (2026-03-29)', () => {
    const r = generateDateRange('2026-03-27', '2026-03-31');
    expect(r).toEqual(['2026-03-27', '2026-03-28', '2026-03-29', '2026-03-30', '2026-03-31']);
  });

  it('has no gaps or duplicates over the autumn DST change (2026-10-25)', () => {
    const r = generateDateRange('2026-10-24', '2026-10-27');
    expect(r).toEqual(['2026-10-24', '2026-10-25', '2026-10-26', '2026-10-27']);
  });

  it('yields 365 / 366 days for a full year', () => {
    expect(generateDateRange('2026-01-01', '2026-12-31')).toHaveLength(365);
    expect(generateDateRange('2024-01-01', '2024-12-31')).toHaveLength(366);
  });
});
