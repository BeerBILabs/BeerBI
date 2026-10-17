import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  formatQuarterLabel,
  getCurrentQuarter,
  getPreviousQuarter,
  getQuarterDates,
  isValidQuarter,
} from '../quarters';

afterEach(() => {
  vi.useRealTimers();
});

describe('getQuarterDates', () => {
  // Regression for audit C6: toISOString() on local midnight shifted dates by one day in UTC+ zones.
  it.each([
    [2026, 1, '2026-01-01', '2026-03-31'],
    [2026, 2, '2026-04-01', '2026-06-30'],
    [2026, 3, '2026-07-01', '2026-09-30'],
    [2026, 4, '2026-10-01', '2026-12-31'],
  ])('year %i Q%i spans %s to %s', (year, quarter, start, end) => {
    expect(getQuarterDates(year, quarter)).toEqual({ start, end });
  });

  it('ends Q1 on Mar 31 in a leap year too', () => {
    expect(getQuarterDates(2024, 1)).toEqual({ start: '2024-01-01', end: '2024-03-31' });
  });
});

describe('getCurrentQuarter', () => {
  it.each([
    ['2026-01-01T12:00:00', 1],
    ['2026-03-31T12:00:00', 1],
    ['2026-04-01T12:00:00', 2],
    ['2026-09-30T12:00:00', 3],
    ['2026-12-31T12:00:00', 4],
  ])('%s is in Q%i', (iso, q) => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date(iso)); // no offset: parsed as local time
    expect(getCurrentQuarter()).toEqual({ year: 2026, quarter: q });
  });
});

describe('getPreviousQuarter', () => {
  it('goes back within a year', () => {
    expect(getPreviousQuarter(2026, 3)).toEqual({ year: 2026, quarter: 2 });
    expect(getPreviousQuarter(2026, 2)).toEqual({ year: 2026, quarter: 1 });
  });

  it('wraps from Q1 to Q4 of the previous year', () => {
    expect(getPreviousQuarter(2026, 1)).toEqual({ year: 2025, quarter: 4 });
  });
});

describe('isValidQuarter', () => {
  it('accepts past and current quarters, rejects future ones', () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-05-10T12:00:00'));
    expect(isValidQuarter(2026, 2)).toBe(true);
    expect(isValidQuarter(2026, 1)).toBe(true);
    expect(isValidQuarter(2026, 3)).toBe(false);
    expect(isValidQuarter(2027, 1)).toBe(false);
    expect(isValidQuarter(2025, 4)).toBe(true);
  });

  it('rejects out-of-range quarters and years', () => {
    expect(isValidQuarter(2026, 0)).toBe(false);
    expect(isValidQuarter(2026, 5)).toBe(false);
    expect(isValidQuarter(2019, 4)).toBe(false);
    expect(isValidQuarter(2100, 1)).toBe(false);
  });
});

describe('formatQuarterLabel', () => {
  it('prefixes Q', () => {
    expect(formatQuarterLabel(3)).toBe('Q3');
  });
});
