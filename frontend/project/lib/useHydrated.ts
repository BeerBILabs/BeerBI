import { useSyncExternalStore } from "react";

const subscribe = () => () => {};

/**
 * False on the server and during hydration, true afterwards.
 *
 * `/` and `/analytics` are prerendered at build time (audit F1), so anything derived from the
 * current date (default ranges, quarter tabs, year lists) would bake the build date into the HTML
 * and break hydration once the quarter or year has changed. Such parts render only when this is
 * true, i.e. from the client's clock after hydration.
 */
export function useHydrated(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => true,
    () => false
  );
}
