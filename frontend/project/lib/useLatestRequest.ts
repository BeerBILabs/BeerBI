import { useEffect, useMemo, useState } from "react";

/** Loads the data for a request key; must honour the signal and be stable (module-level). */
export type RequestLoader<T> = (key: string, signal: AbortSignal) => Promise<T>;

/** One request run: a new object whenever `key` or `load` changes, also when a key comes back. */
type Run<T> = { key: string | null; load: RequestLoader<T> };

type State<T> = {
  /** The run that finished most recently (successful or not). */
  run: Run<T>;
  error: string | null;
  /** Last successful result and the key it belongs to; kept while a newer request runs. */
  data: T | null;
  dataKey: string | null;
};

export type LatestRequest<T> = {
  /** Last successful result, possibly for an older key while `isFetching` (stale-while-revalidate). */
  data: T | null;
  /** Key the `data` belongs to. */
  dataKey: string | null;
  /** A request for the current key is in flight. */
  isFetching: boolean;
  /** Error of the request for the current key. */
  error: string | null;
};

/**
 * Runs `load(key)` whenever `key` changes and keeps the previous result visible until the new
 * one arrives. A superseded request is aborted, so only the latest key can update the state.
 * `key === null` disables loading.
 *
 * "Settled" is measured per run, not per key: after a → b → a the second request for `a` is
 * reported as fetching, and an old error of `a` is not shown while it runs.
 */
export function useLatestRequest<T>(key: string | null, load: RequestLoader<T>): LatestRequest<T> {
  const run = useMemo<Run<T>>(() => ({ key, load }), [key, load]);
  const [state, setState] = useState<State<T> | null>(null);

  useEffect(() => {
    const { key, load } = run;
    if (key === null) return;
    const controller = new AbortController();
    load(key, controller.signal).then(
      (data) => {
        if (controller.signal.aborted) return;
        setState({ run, error: null, data, dataKey: key });
      },
      (err: unknown) => {
        if (controller.signal.aborted) return;
        const message = err instanceof Error ? err.message : "Unknown error";
        setState((prev) => ({ run, error: message, data: prev?.data ?? null, dataKey: prev?.dataKey ?? null }));
      }
    );
    return () => controller.abort();
  }, [run]);

  const settled = state !== null && state.run === run;
  return {
    data: state?.data ?? null,
    dataKey: state?.dataKey ?? null,
    isFetching: key !== null && !settled,
    error: settled ? state.error : null,
  };
}
