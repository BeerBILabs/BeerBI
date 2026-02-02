"use client";

import type { ReactNode } from "react";
import { usePathname } from "next/navigation";
import { RankingsNav } from "@/components/RankingsNav";
import { getCurrentQuarter } from "@/lib/quarters";

interface RankingsLayoutProps {
  children: ReactNode;
}

export default function RankingsLayout({ children }: RankingsLayoutProps) {
  const pathname = usePathname();
  const currentQuarter = getCurrentQuarter();

  // Parse pathname to determine active tab and year
  // /rankings/all -> activeTab = 'all', activeYear = current year (ignored)
  // /rankings/YYYY/qN -> activeTab = 'qN', activeYear = YYYY
  let activeTab: "all" | "q1" | "q2" | "q3" | "q4" = "all";
  let activeYear = currentQuarter.year;

  if (pathname === "/rankings/all") {
    activeTab = "all";
    activeYear = currentQuarter.year;
  } else {
    // Match /rankings/YYYY/qN pattern
    const match = pathname.match(/\/rankings\/(\d{4})\/q([1-4])/i);
    if (match) {
      activeYear = parseInt(match[1], 10);
      const quarterNum = parseInt(match[2], 10);
      activeTab = `q${quarterNum}` as "q1" | "q2" | "q3" | "q4";
    } else {
      // Default to current quarter (shouldn't happen due to redirect)
      activeTab = `q${currentQuarter.quarter}` as "q1" | "q2" | "q3" | "q4";
      activeYear = currentQuarter.year;
    }
  }

  return (
    <div
      className="w-full max-w-4xl mx-auto px-4 py-6"
      style={{ backgroundColor: "hsl(var(--background))" }}
    >
      <h1
        className="text-2xl font-bold mb-6"
        style={{ color: "hsl(var(--primary))" }}
      >
        Rankings
      </h1>
      <RankingsNav activeTab={activeTab} activeYear={activeYear} />
      <div className="mt-6">{children}</div>
    </div>
  );
}
