import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useAnalyticsData } from '../useAnalyticsData';

type Call = { url: string; signal: AbortSignal; resolve: (body: unknown, ok?: boolean) => void };
const calls: Call[] = [];

function data(tag: string) {
  return { timeline: [{ date: tag, count: 1 }], top_givers: [], top_recipients: [], heatmap: [], pairs: [] };
}

beforeEach(() => {
  calls.length = 0;
  vi.stubGlobal('fetch', vi.fn((url: string, init: RequestInit) =>
    new Promise((resolve) => {
      calls.push({
        url,
        signal: init.signal as AbortSignal,
        resolve: (body, ok = true) => resolve({ ok, statusText: ok ? 'OK' : 'Internal Server Error', json: async () => body }),
      });
    })
  ));
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('useAnalyticsData', () => {
  it('requests /stats/combined with all parameters', () => {
    renderHook(() => useAnalyticsData({ startDate: '2026-01-01', endDate: '2026-10-18', granularity: 'week', limit: 10, pairsLimit: 15 }));
    expect(calls[0].url).toBe('/api/proxy/stats/combined?start=2026-01-01&end=2026-10-18&granularity=week&limit=10&pairs_limit=15');
  });

  it('keeps the previous data and its range while a new range loads (F6)', async () => {
    const { result, rerender } = renderHook(
      ({ start }) => useAnalyticsData({ startDate: start, endDate: '2026-10-18' }),
      { initialProps: { start: '2026-01-01' } }
    );
    expect(result.current.loading).toBe(true);
    await act(async () => calls[0].resolve(data('first')));
    expect(result.current).toMatchObject({ loading: false, isFetching: false, dataRange: { start: '2026-01-01', end: '2026-10-18' } });

    rerender({ start: '2026-07-01' });
    expect(result.current.loading).toBe(false);
    expect(result.current.isFetching).toBe(true);
    expect(result.current.data?.timeline[0].date).toBe('first');
    expect(result.current.dataRange).toEqual({ start: '2026-01-01', end: '2026-10-18' });

    rerender({ start: '2026-04-01' });
    expect(calls[1].signal.aborted).toBe(true);

    await act(async () => calls[2].resolve(data('third')));
    expect(result.current.data?.timeline[0].date).toBe('third');
    expect(result.current.dataRange).toEqual({ start: '2026-04-01', end: '2026-10-18' });
  });

  it('reports HTTP errors', async () => {
    const { result } = renderHook(() => useAnalyticsData({ startDate: '2026-01-01', endDate: '2026-10-18' }));
    await act(async () => calls[0].resolve({}, false));
    expect(result.current.error).toBe('Failed to fetch analytics data: Internal Server Error');
    expect(result.current.loading).toBe(false);
  });

  it('does not fetch when disabled', () => {
    const { result } = renderHook(() => useAnalyticsData({ startDate: '2026-01-01', endDate: '2026-10-18', enabled: false }));
    expect(calls).toHaveLength(0);
    expect(result.current.loading).toBe(false);
  });
});
