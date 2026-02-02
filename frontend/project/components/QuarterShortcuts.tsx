"use client";

import Link from "next/link";

interface QuarterShortcutsProps {
  currentQuarter: { year: number; quarter: number };
  lastQuarter: { year: number; quarter: number };
  activeYear: number;
  activeQuarter: number | "all";
}

export function QuarterShortcuts({
  currentQuarter,
  lastQuarter,
  activeYear,
  activeQuarter,
}: QuarterShortcutsProps) {
  const isCurrentActive =
    activeQuarter !== "all" &&
    activeYear === currentQuarter.year &&
    activeQuarter === currentQuarter.quarter;

  const isLastActive =
    activeQuarter !== "all" &&
    activeYear === lastQuarter.year &&
    activeQuarter === lastQuarter.quarter;

  return (
    <div className="flex gap-2 mb-4">
      <Link
        href={`/rankings/${currentQuarter.year}/q${currentQuarter.quarter}`}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors"
        style={{
          backgroundColor: isCurrentActive
            ? "hsl(var(--primary))"
            : "hsl(var(--secondary))",
          color: isCurrentActive
            ? "hsl(var(--primary-foreground))"
            : "hsl(var(--secondary-foreground))",
        }}
      >
        Current Quarter
      </Link>
      <Link
        href={`/rankings/${lastQuarter.year}/q${lastQuarter.quarter}`}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors"
        style={{
          backgroundColor: isLastActive
            ? "hsl(var(--primary))"
            : "hsl(var(--secondary))",
          color: isLastActive
            ? "hsl(var(--primary-foreground))"
            : "hsl(var(--secondary-foreground))",
        }}
      >
        Last Quarter
      </Link>
    </div>
  );
}
