"use client";

import { ChevronUp, ChevronDown, ChevronsUp, ChevronsDown, Minus } from "lucide-react";

interface RankChangeIndicatorProps {
  currentRank: number;
  previousRank: number | null;
}

/**
 * Displays rank movement compared to previous quarter.
 * - Single arrow for 1-2 place changes
 * - Double arrows for 3+ place changes
 * - NEW badge for users not in previous quarter
 * - Minus icon for no change
 */
export function RankChangeIndicator({
  currentRank,
  previousRank,
}: RankChangeIndicatorProps) {
  // New to leaderboard this quarter
  if (previousRank === null) {
    return (
      <span
        className="text-xs font-medium px-1.5 py-0.5 rounded"
        style={{
          backgroundColor: "hsl(var(--accent))",
          color: "hsl(var(--accent-foreground))",
        }}
        aria-label="New this quarter"
      >
        NEW
      </span>
    );
  }

  // Calculate change: positive = moved up (lower rank number is better)
  const change = previousRank - currentRank;

  // No change
  if (change === 0) {
    return (
      <Minus
        className="w-4 h-4"
        style={{ color: "hsl(var(--muted-foreground))" }}
        aria-label="No change"
      />
    );
  }

  // Moved up (positive change)
  if (change > 0) {
    const Icon = change >= 3 ? ChevronsUp : ChevronUp;
    return (
      <Icon
        className="w-4 h-4"
        style={{ color: "hsl(142 76% 36%)" }}
        aria-label={`Up ${change} place${change !== 1 ? "s" : ""}`}
      />
    );
  }

  // Moved down (negative change)
  const absChange = Math.abs(change);
  const Icon = absChange >= 3 ? ChevronsDown : ChevronDown;
  return (
    <Icon
      className="w-4 h-4"
      style={{ color: "hsl(0 72% 51%)" }}
      aria-label={`Down ${absChange} place${absChange !== 1 ? "s" : ""}`}
    />
  );
}
