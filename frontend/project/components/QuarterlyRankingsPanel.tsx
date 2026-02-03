"use client";

import { useState } from "react";
import type { ReactElement } from "react";
import { Leaderboard } from "./Leaderboard";
import { YearSelector } from "./YearSelector";
import { getCurrentQuarter, getPreviousQuarter } from "@/lib/quarters";

type ActiveTab = "all" | "q1" | "q2" | "q3" | "q4";

interface QuarterShortcutsInlineProps {
  currentQuarter: { year: number; quarter: number };
  lastQuarter: { year: number; quarter: number };
  activeYear: number;
  activeQuarter: number | "all";
  onSelect: (year: number, quarter: number) => void;
}

function QuarterShortcutsInline({
  currentQuarter,
  lastQuarter,
  activeYear,
  activeQuarter,
  onSelect,
}: QuarterShortcutsInlineProps) {
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
      <button
        type="button"
        onClick={() => onSelect(currentQuarter.year, currentQuarter.quarter)}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors cursor-pointer"
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
      </button>
      <button
        type="button"
        onClick={() => onSelect(lastQuarter.year, lastQuarter.quarter)}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors cursor-pointer"
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
      </button>
    </div>
  );
}

export function QuarterlyRankingsPanel(): ReactElement {
  const currentQuarter = getCurrentQuarter();
  const lastQuarter = getPreviousQuarter(currentQuarter.year, currentQuarter.quarter);

  const [activeTab, setActiveTab] = useState<ActiveTab>(
    `q${currentQuarter.quarter}` as ActiveTab
  );
  const [activeYear, setActiveYear] = useState<number>(currentQuarter.year);

  // Determine which quarter tabs to show
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
    return [1];
  };

  const visibleQuarters = getVisibleQuarters();

  const handleYearChange = (newYear: number) => {
    if (activeTab === "all") {
      return;
    }
    const quarterNum = parseInt(activeTab.slice(1), 10);
    if (newYear === currentQuarter.year && quarterNum > currentQuarter.quarter) {
      setActiveTab(`q${currentQuarter.quarter}` as ActiveTab);
    }
    setActiveYear(newYear);
  };

  const handleShortcutSelect = (year: number, quarter: number) => {
    setActiveYear(year);
    setActiveTab(`q${quarter}` as ActiveTab);
  };

  const handleTabClick = (tab: ActiveTab) => {
    setActiveTab(tab);
  };

  // Build tabs array
  const tabs: Array<{ id: ActiveTab; label: string }> = [
    { id: "all", label: "All Time" },
  ];

  for (const q of visibleQuarters) {
    tabs.push({
      id: `q${q}` as ActiveTab,
      label: `Q${q}`,
    });
  }

  // Get active quarter number for shortcuts
  const activeQuarterNum: number | "all" =
    activeTab === "all" ? "all" : parseInt(activeTab.slice(1), 10);

  // Determine year/quarter props for Leaderboard
  const leaderboardYear = activeTab === "all" ? null : activeYear;
  const leaderboardQuarter = activeTab === "all" ? null : parseInt(activeTab.slice(1), 10);

  return (
    <section>
      <h1
        className="text-3xl font-extrabold mb-2 tracking-tight"
        style={{ color: "hsl(var(--primary))" }}
      >
        Quarterly Rankings
      </h1>
      <p className="mb-6 text-base" style={{ color: "hsl(var(--muted-foreground))" }}>
        Compare performance across quarters with{" "}
        <span className="font-semibold" style={{ color: "hsl(var(--primary))" }}>
          rank changes
        </span>
        .
      </p>

      <QuarterShortcutsInline
        currentQuarter={currentQuarter}
        lastQuarter={lastQuarter}
        activeYear={activeYear}
        activeQuarter={activeQuarterNum}
        onSelect={handleShortcutSelect}
      />

      <div className="flex items-center gap-4 mb-4">
        <YearSelector
          selectedYear={activeYear}
          onChange={handleYearChange}
          disabled={activeTab === "all"}
        />
        <nav role="tablist" aria-label="Rankings period" className="flex gap-1">
          {tabs.map((tab) => {
            const isActive = tab.id === activeTab;
            return (
              <button
                key={tab.id}
                type="button"
                role="tab"
                aria-selected={isActive}
                onClick={() => handleTabClick(tab.id)}
                className="px-4 py-2 rounded-t-md text-sm font-medium transition-colors text-center cursor-pointer"
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
                  minWidth: "4rem",
                }}
              >
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      <div className="animate-fadein">
        <Leaderboard
          key={activeTab === "all" ? "all-time" : `${activeYear}-${activeTab}`}
          year={leaderboardYear}
          quarter={leaderboardQuarter}
          showRankChange={activeTab !== "all"}
          maxHeight=""
        />
      </div>
    </section>
  );
}
