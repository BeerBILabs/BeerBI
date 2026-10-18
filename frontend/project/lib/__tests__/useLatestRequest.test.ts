import { describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { useLatestRequest, type RequestLoader } from '../useLatestRequest';

type Pending = { key: string; signal: AbortSignal; resolve: (v: string) => void; reject: (e: unknown) => void };

function controllableLoader() {
  const calls: Pending[] = [];
  const load: RequestLoader<string> = (key, signal) =>
    new Promise<string>((resolve, reject) => calls.push({ key, signal, resolve, reject }));
  return { load, calls };
}

describe('useLatestRequest', () => {
  it('loads the key and reports fetching until the result arrives', async () => {
    const { load, calls } = controllableLoader();
    const { result } = renderHook(() => useLatestRequest('a', load));

    expect(result.current).toEqual({ data: null, dataKey: null, isFetching: true, error: null });
    await act(async () => calls[0].resolve('A'));
    expect(result.current).toEqual({ data: 'A', dataKey: 'a', isFetching: false, error: null });
  });

  it('keeps the previous data while the next key loads and aborts superseded requests', async () => {
    const { load, calls } = controllableLoader();
    const { result, rerender } = renderHook(({ k }) => useLatestRequest(k, load), { initialProps: { k: 'a' } });
    await act(async () => calls[0].resolve('A'));

    rerender({ k: 'b' });
    expect(result.current).toMatchObject({ data: 'A', dataKey: 'a', isFetching: true });

    rerender({ k: 'c' });
    expect(calls[1].signal.aborted).toBe(true);
    // a late answer for the aborted key changes nothing
    await act(async () => calls[1].resolve('B'));
    expect(result.current).toMatchObject({ data: 'A', isFetching: true });

    await act(async () => calls[2].resolve('C'));
    expect(result.current).toEqual({ data: 'C', dataKey: 'c', isFetching: false, error: null });
  });

  it('reports an error for the current key and keeps the last good data', async () => {
    const { load, calls } = controllableLoader();
    const { result, rerender } = renderHook(({ k }) => useLatestRequest(k, load), { initialProps: { k: 'a' } });
    await act(async () => calls[0].resolve('A'));

    rerender({ k: 'b' });
    await act(async () => calls[1].reject(new Error('boom')));
    expect(result.current).toEqual({ data: 'A', dataKey: 'a', isFetching: false, error: 'boom' });

    rerender({ k: 'c' });
    expect(result.current.error).toBeNull();
    await act(async () => calls[2].reject('not an error'));
    expect(result.current.error).toBe('Unknown error');
  });

  it('does nothing for a null key', () => {
    const load = vi.fn<RequestLoader<string>>();
    const { result } = renderHook(() => useLatestRequest(null, load));
    expect(load).not.toHaveBeenCalled();
    expect(result.current.isFetching).toBe(false);
  });

  it('aborts on unmount', async () => {
    const { load, calls } = controllableLoader();
    const { unmount } = renderHook(() => useLatestRequest('a', load));
    await waitFor(() => expect(calls).toHaveLength(1));
    unmount();
    expect(calls[0].signal.aborted).toBe(true);
  });
});

describe('useLatestRequest when a key comes back', () => {
  it('reports the refetch of an earlier key as fetching and hides its old error (a -> b -> a)', async () => {
    const { load, calls } = controllableLoader();
    const { result, rerender } = renderHook(({ k }) => useLatestRequest(k, load), { initialProps: { k: 'a' } });
    await act(async () => calls[0].reject(new Error('old failure')));
    expect(result.current.error).toBe('old failure');

    rerender({ k: 'b' });
    rerender({ k: 'a' });
    expect(calls).toHaveLength(3);
    expect(calls[1].signal.aborted).toBe(true);
    expect(result.current).toMatchObject({ isFetching: true, error: null });

    await act(async () => calls[2].resolve('A'));
    expect(result.current).toEqual({ data: 'A', dataKey: 'a', isFetching: false, error: null });
  });

  it('starts no new request on a re-render with the same key', async () => {
    const { load, calls } = controllableLoader();
    const { result, rerender } = renderHook(({ k }) => useLatestRequest(k, load), { initialProps: { k: 'a' } });
    await act(async () => calls[0].resolve('A'));

    rerender({ k: 'a' });
    rerender({ k: 'a' });
    expect(calls).toHaveLength(1);
    expect(result.current).toEqual({ data: 'A', dataKey: 'a', isFetching: false, error: null });
  });
});
