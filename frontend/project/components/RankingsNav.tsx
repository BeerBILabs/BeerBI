"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { getCurrentQuarter, getPreviousQuarter } from "@/lib/quarters";
import { YearSelector } from "./YearSelector";
import { QuarterShortcuts } from "./QuarterShortcuts";

interface RankingsNavProps {
  activeTab: "all" | "q1" | "q2" | "q3" | "q4";
  activeYear: number;
}

export function RankingsNav({ activeTab, activeYear }: RankingsNavProps) {
  const router = useRouter();
  const currentQuarter = getCurrentQuarter();
  const lastQuarter = getPreviousQuarter(
    currentQuarter.year,
    currentQuarter.quarter
  );

  // Determine which quarter tabs to show
  // For current year: only show quarters up to and including current quarter
  // For past years: show all 4 quarters
  const getVisibleQuarters = (): number[] => {
    if (activeYear < currentQuarter.year) {
      return [1, 2, 3, 4];
    }
    if (activeYear === currentQuarter.year) {
      const quarters: number[] = [];
      for (let q = 1; q <= currentQuarter.quarter; q++) {
        quarters.push(q);
      }
      return quarters;
    }
    // Future year - shouldn't happen but return current quarter only
    return [1];
  };

  const visibleQuarters = getVisibleQuarters();

  const handleYearChange = (newYear: number) => {
    if (activeTab === "all") {
      // All Time ignores year selector, do nothing
      return;
    }
    // Navigate to same quarter, new year
    const quarterNum = parseInt(activeTab.slice(1), 10);
    // If navigating to current year and quarter is in the future, go to current quarter
    if (
      newYear === currentQuarter.year &&
      quarterNum > currentQuarter.quarter
    ) {
      router.push(`/rankings/${newYear}/q${currentQuarter.quarter}`);
    } else {
      router.push(`/rankings/${newYear}/q${quarterNum}`);
    }
  };

  // Build tabs array
  const tabs: Array<{ id: "all" | "q1" | "q2" | "q3" | "q4"; label: string; href: string }> = [
    { id: "all", label: "All Time", href: "/rankings/all" },
  ];

  for (const q of visibleQuarters) {
    tabs.push({
      id: `q${q}` as "q1" | "q2" | "q3" | "q4",
      label: `Q${q}`,
      href: `/rankings/${activeYear}/q${q}`,
    });
  }

  // Get active quarter number for shortcuts
  const activeQuarterNum: number | "all" =
    activeTab === "all" ? "all" : parseInt(activeTab.slice(1), 10);

  return (
    <div className="mb-6">
      <QuarterShortcuts
        currentQuarter={currentQuarter}
        lastQuarter={lastQuarter}
        activeYear={activeYear}
        activeQuarter={activeQuarterNum}
      />
      <div className="flex items-center gap-4">
        <YearSelector
          selectedYear={activeYear}
          onChange={handleYearChange}
          disabled={activeTab === "all"}
        />
        <nav role="tablist" aria-label="Rankings period" className="flex gap-1">
          {tabs.map((tab) => {
            const isActive = tab.id === activeTab;
            return (
              <Link
                key={tab.id}
                href={tab.href}
                role="tab"
                aria-selected={isActive}
                className="px-4 py-2 rounded-t-md text-sm font-medium transition-colors"
                style={{
                  backgroundColor: isActive
                    ? "hsl(var(--primary))"
                    : "transparent",
                  color: isActive
                    ? "hsl(var(--primary-foreground))"
                    : "hsl(var(--muted-foreground))",
                  borderBottom: isActive
                    ? "2px solid hsl(var(--primary))"
                    : "2px solid transparent",
                }}
              >
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </div>
    </div>
  );
}
