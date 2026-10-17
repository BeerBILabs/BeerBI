"use client";

import type { ReactNode } from "react";

interface ChartContainerProps {
  title: string;
  subtitle?: string;
  children: ReactNode;
  loading?: boolean;
  error?: string | null;
  onRetry?: () => void;
  className?: string;
  /** Tailwind height of the loading placeholder; match the chart to avoid layout shift. */
  skeletonHeight?: string;
}

function LoadingSkeleton({ heightClass }: { heightClass: string }) {
  return (
    <div className="animate-pulse">
      <div
        className={`${heightClass} rounded-lg`}
        style={{ backgroundColor: "hsl(var(--muted))" }}
      />
    </div>
  );
}

export function ChartContainer({
  title,
  subtitle,
  children,
  loading = false,
  error = null,
  onRetry,
  className = "",
  skeletonHeight = "h-64",
}: ChartContainerProps) {
  return (
    <div
      className={`p-5 rounded-2xl shadow-lg border ${className}`}
      style={{
        backgroundColor: "hsl(var(--card))",
        borderColor: "hsl(var(--border))",
      }}
    >
      <h3
        className="text-lg font-bold mb-1"
        style={{ color: "hsl(var(--primary))" }}
      >
        {title}
      </h3>
      {subtitle && (
        <p
          className="text-sm mb-4"
          style={{ color: "hsl(var(--muted-foreground))" }}
        >
          {subtitle}
        </p>
      )}

      {loading ? (
        <LoadingSkeleton heightClass={skeletonHeight} />
      ) : error ? (
        <div className="text-center py-8">
          <p
            className="text-sm mb-3"
            style={{ color: "hsl(var(--destructive))" }}
          >
            {error}
          </p>
          {onRetry && (
            <button
              onClick={onRetry}
              className="px-3 py-1.5 rounded text-sm font-medium transition-colors"
              style={{
                backgroundColor: "hsl(var(--primary))",
                color: "hsl(var(--primary-foreground))",
              }}
            >
              Retry
            </button>
          )}
        </div>
      ) : (
        children
      )}
    </div>
  );
}

/** Placeholder shown while a lazily loaded chart chunk is on its way (same card as the real chart). */
export function ChartSkeleton({
  title,
  subtitle,
  skeletonHeight,
}: {
  title: string;
  subtitle?: string;
  skeletonHeight?: string;
}) {
  return (
    <ChartContainer title={title} subtitle={subtitle} skeletonHeight={skeletonHeight} loading>
      {null}
    </ChartContainer>
  );
}
