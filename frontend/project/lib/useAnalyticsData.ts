import { useState, useEffect } from "react";

type TimelinePoint = {
  date: string;
  given: number;
  received: number;
};

type TopUserStats = {
  userId: string; // Backend uses camelCase
  count: number;
};

type HeatmapPoint = {
  date: string;
  count: number;
};

type PairStats = {
  giver: string;
  recipient: string;
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

export function useAnalyticsData({
  startDate,
  endDate,
  granularity = "day",
  limit = 20,
  pairsLimit = 15,
  enabled = true,
}: UseAnalyticsDataOptions) {
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    async function fetchData() {
      try {
        setLoading(true);
        setError(null);

        const params = new URLSearchParams({
          start: startDate,
          end: endDate,
          granularity,
          limit: limit.toString(),
          pairs_limit: pairsLimit.toString(),
        });

        const resp = await fetch(`/api/proxy/stats/combined?${params}`);
        if (!resp.ok) {
          throw new Error(`Failed to fetch analytics data: ${resp.statusText}`);
        }

        const result = await resp.json();
        if (!cancelled) {
          setData(result);
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Analytics data fetch error:", err);
          setError(err instanceof Error ? err.message : "Unknown error");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchData();

    return () => {
      cancelled = true;
    };
  }, [startDate, endDate, granularity, limit, pairsLimit, enabled]);

  return { data, loading, error };
}
