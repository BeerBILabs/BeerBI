"use client";

import { useEffect, useMemo, useState } from "react";
import { ChartContainer } from "./ChartContainer";
import { defaultChartColors } from "@/lib/chartTheme";
import { knownUserNames, pairUserRefs, resolveUsers } from "@/lib/userNames";
import type { PairStats } from "@/lib/statsTypes";

interface NetworkChartProps {
  startDate: string;
  endDate: string;
  limit?: number;
  /** Rows from /stats/combined; names are taken from them when the backend sends them inline. */
  preloadedData?: PairStats[];
}

export function NetworkChart({ startDate, endDate, limit = 20, preloadedData }: NetworkChartProps) {
  const [data, setData] = useState<PairStats[]>([]);
  const [names, setNames] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(!preloadedData);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    async function fetchData() {
      setError(null);
      try {
        let pairs: PairStats[];

        if (preloadedData) {
          // Preloaded rows are shown at once; the list stays mounted while names resolve.
          pairs = preloadedData;
        } else {
          setLoading(true);
          const params = new URLSearchParams({
            start: startDate,
            end: endDate,
            limit: limit.toString(),
          });
          const resp = await fetch(`/api/proxy/stats/pairs?${params}`, { signal: controller.signal });
          if (!resp.ok) throw new Error("Failed to fetch pair data");
          const json = await resp.json();
          pairs = json || [];
        }
        if (controller.signal.aborted) return;
        setData(pairs);

        if (pairs.length > 0) {
          const resolved = await resolveUsers(pairUserRefs(pairs));
          if (controller.signal.aborted) return;
          setNames(Object.fromEntries(Object.entries(resolved).map(([id, u]) => [id, u.name])));
        }
      } catch (err) {
        if (controller.signal.aborted) return;
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }
    fetchData();

    return () => controller.abort();
  }, [startDate, endDate, limit, preloadedData]);

  const colors = defaultChartColors;
  // Names known without a request (inline or cached) label new rows at once; `names` may still
  // belong to the previous rows while resolveUsers runs.
  const quickNames = useMemo(() => knownUserNames(pairUserRefs(data)), [data]);

  // Calculate max count for scaling
  const maxCount = Math.max(...data.map((d) => d.count), 1);

  if (!loading && data.length === 0) {
    return (
      <ChartContainer
        title="Who Appreciates Whom"
        subtitle="Giver → Recipient relationships"
        loading={false}
        error={null}
      >
        <div className="h-64 flex items-center justify-center">
          <p style={{ color: colors.foreground }} className="text-sm opacity-70">
            No pair data available for this period
          </p>
        </div>
      </ChartContainer>
    );
  }

  return (
    <ChartContainer
      title="🤝 Who Appreciates Whom"
      subtitle="Top giver → recipient pairs"
      loading={loading}
      error={error}
    >
      <div className="space-y-2 max-h-80 overflow-y-auto pr-2">
        {data.map((pair, index) => {
          const giverName = quickNames[pair.giver] || names[pair.giver] || pair.giver;
          const recipientName = quickNames[pair.recipient] || names[pair.recipient] || pair.recipient;
          const widthPercent = (pair.count / maxCount) * 100;

          return (
            <div key={`${pair.giver}-${pair.recipient}`} className="flex items-center gap-2">
              <div
                className="text-sm font-medium w-30 truncate text-right"
                style={{ color: "hsl(var(--foreground))" }}
                title={giverName}
              >
                {giverName}
              </div>
              <div className="w-40 relative h-5">
                <div
                  className="absolute inset-y-0 left-0 rounded-r flex items-center justify-end pr-1.5"
                  style={{
                    width: `${Math.max(widthPercent, 50)}%`,
                    backgroundColor: colors.given,
                    opacity: 0.9 - index * 0.03,
                  }}
                >
                  <span className="text-xs font-bold" style={{ color: "hsl(var(--primary-foreground))" }}>
                    {pair.count}
                  </span>
                </div>
              </div>
              <div
                className="text-sm font-medium w-30 truncate text-left"
                style={{ color: "hsl(var(--foreground))" }}
                title={recipientName}
              >
                {recipientName}
              </div>
            </div>
          );
        })}
      </div>
    </ChartContainer>
  );
}
