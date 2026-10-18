import { useLatestRequest } from "./useLatestRequest";
import type { PairStats, TopUserStats } from "./statsTypes";

type TimelinePoint = {
  date: string;
  count: number; // beers given in the period
};

type HeatmapPoint = {
  date: string;
  count: number;
};

export type AnalyticsData = {
  timeline: TimelinePoint[];
  top_givers: TopUserStats[];
  top_recipients: TopUserStats[];
  heatmap: HeatmapPoint[];
  pairs: PairStats[];
};

type UseAnalyticsDataOptions = {
  startDate: string;
  endDate: string;
  granularity?: "day" | "week" | "month";
  limit?: number;
  pairsLimit?: number;
  enabled?: boolean;
};

/** Result of one /stats/combined request together with the range it was requested for. */
type LoadedAnalytics = { data: AnalyticsData; start: string; end: string };

const COMBINED_PATH = "/api/proxy/stats/combined";

async function loadAnalytics(url: string, signal: AbortSignal): Promise<LoadedAnalytics> {
  const resp = await fetch(url, { signal });
  if (!resp.ok) {
    throw new Error(`Failed to fetch analytics data: ${resp.statusText}`);
  }
  const data = (await resp.json()) as AnalyticsData;
  const params = new URL(url, "http://localhost").searchParams;
  return { data, start: params.get("start") ?? "", end: params.get("end") ?? "" };
}

/**
 * All analytics data of a range in one request (/stats/combined).
 *
 * While a new range loads, the previous `data` (and its `dataRange`) stays available so the
 * charts stay mounted (audit F6); `isFetching` tells that a newer request is running. A
 * superseded request is aborted.
 */
export function useAnalyticsData({
  startDate,
  endDate,
  granularity = "day",
  limit = 20,
  pairsLimit = 15,
  enabled = true,
}: UseAnalyticsDataOptions) {
  const params = new URLSearchParams({
    start: startDate,
    end: endDate,
    granularity,
    limit: limit.toString(),
    pairs_limit: pairsLimit.toString(),
  });
  const key = enabled ? `${COMBINED_PATH}?${params}` : null;
  const { data: loaded, isFetching, error } = useLatestRequest(key, loadAnalytics);

  return {
    data: loaded?.data ?? null,
    /** The range `data` belongs to (lags behind the requested range while fetching). */
    dataRange: loaded ? { start: loaded.start, end: loaded.end } : null,
    isFetching,
    /** True only before the first result: nothing to show yet. */
    loading: isFetching && loaded === null,
    error,
  };
}
