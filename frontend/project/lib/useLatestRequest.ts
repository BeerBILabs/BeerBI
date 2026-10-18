import { useEffect, useState } from "react";

/** Loads the data for a request key; must honour the signal and be stable (module-level). */
export type RequestLoader<T> = (key: string, signal: AbortSignal) => Promise<T>;

/**
 * One request run. `id` is an explicit counter that grows whenever `key` or `load` changes, also
 * when a key comes back (a → b → a gives three runs).
 */
type Run<T> = { id: number; key: string | null; load: RequestLoader<T> };

type State<T> = {
  /** Id of the run that finished most recently (successful or not). */
  runId: number;
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
  // The run counter lives in state, not in a memo: React may drop a memo cache, which would
  // start a new request (and report "fetching") without any input change.
  const [run, setRun] = useState<Run<T>>(() => ({ id: 0, key, load }));
  if (run.key !== key || run.load !== load) {
    // Adjusting state while rendering: React re-renders before committing.
    setRun({ id: run.id + 1, key, load });
  }
  const [state, setState] = useState<State<T> | null>(null);

  useEffect(() => {
    const { id, key, load } = run;
    if (key === null) return;
    const controller = new AbortController();
    load(key, controller.signal).then(
      (data) => {
        if (controller.signal.aborted) return;
        setState({ runId: id, error: null, data, dataKey: key });
      },
      (err: unknown) => {
        if (controller.signal.aborted) return;
        const message = err instanceof Error ? err.message : "Unknown error";
        setState((prev) => ({ runId: id, error: message, data: prev?.data ?? null, dataKey: prev?.dataKey ?? null }));
      }
    );
    return () => controller.abort();
  }, [run]);

  const settled = state !== null && state.runId === run.id;
  return {
    data: state?.data ?? null,
    dataKey: state?.dataKey ?? null,
    isFetching: key !== null && !settled,
    error: settled ? state.error : null,
  };
}
