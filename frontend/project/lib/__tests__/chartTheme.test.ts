import { afterEach, describe, expect, it } from 'vitest';
import { defaultChartColors, formatDate, formatNumber, getChartColors } from '../chartTheme';

describe('formatNumber', () => {
  it.each([
    [0, '0'],
    [999, '999'],
    [1000, '1.0K'],
    [1500, '1.5K'],
    [999999, '1000.0K'],
    [1000000, '1.0M'],
    [2500000, '2.5M'],
  ])('%i -> %s', (input, expected) => {
    expect(formatNumber(input)).toBe(expected);
  });
});

describe('formatDate', () => {
  it('formats day granularity as "Mon D" in local time', () => {
    expect(formatDate('2026-01-15')).toBe('Jan 15');
    expect(formatDate('2026-03-01', 'day')).toBe('Mar 1');
    expect(formatDate('2026-12-31', 'day')).toBe('Dec 31');
  });

  it('formats week granularity as "Wnn YYYY"', () => {
    expect(formatDate('2026-W01', 'week')).toBe('2026 W01');
  });

  it('formats month granularity as "Mon YYYY"', () => {
    expect(formatDate('2026-01', 'month')).toBe('Jan 2026');
    expect(formatDate('2026-12', 'month')).toBe('Dec 2026');
  });
});

describe('getChartColors', () => {
  afterEach(() => {
    document.documentElement.style.cssText = '';
  });

  it('returns empty strings when CSS variables are not defined', () => {
    const c = getChartColors();
    expect(c.primary).toBe('');
    expect(c.heatmap).toHaveLength(5);
  });

  it('wraps CSS variable values in hsl()', () => {
    document.documentElement.style.setProperty('--primary', '40 70% 55%');
    const c = getChartColors();
    expect(c.primary).toBe('hsl(40 70% 55%)');
    expect(c.given).toBe(c.primary);
    expect(c.heatmap[4]).toBe(c.primary);
  });
});

describe('defaultChartColors', () => {
  it('has five heatmap levels', () => {
    expect(defaultChartColors.heatmap).toHaveLength(5);
  });
});
