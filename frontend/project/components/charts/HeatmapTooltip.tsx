"use client";

import { useImperativeHandle, useState, type Ref } from "react";
import { parseLocalDate } from "@/lib/dateUtils";

export interface HeatmapTooltipHandle {
  show: (el: HTMLElement, date: string, count: number) => void;
  hide: () => void;
}

interface TooltipState {
  visible: boolean;
  x: number;
  y: number;
  date: string;
  count: number;
}

const HIDDEN: TooltipState = { visible: false, x: 0, y: 0, date: "", count: 0 };

/**
 * Owns the tooltip state, so moving over cells re-renders only this small component.
 * The parent drives it through a ref (`show`/`hide`) and never re-renders itself.
 */
export function HeatmapTooltip({ ref }: { ref: Ref<HeatmapTooltipHandle> }) {
  const [tooltip, setTooltip] = useState<TooltipState>(HIDDEN);

  useImperativeHandle(
    ref,
    () => ({
      show(el, date, count) {
        const rect = el.getBoundingClientRect();
        setTooltip({ visible: true, x: rect.left + rect.width / 2, y: rect.top - 8, date, count });
      },
      hide() {
        setTooltip((prev) => (prev.visible ? { ...prev, visible: false } : prev));
      },
    }),
    []
  );

  if (!tooltip.visible) return null;

  // Positioned fixed to escape the overflow container
  return (
    <div
      role="tooltip"
      className="fixed z-50 px-3 py-2 text-xs font-medium rounded-lg shadow-lg pointer-events-none"
      style={{
        left: tooltip.x,
        top: tooltip.y,
        transform: "translate(-50%, -100%)",
        backgroundColor: "hsl(var(--card))",
        color: "hsl(var(--card-foreground))",
        border: "1px solid hsl(var(--border))",
      }}
    >
      <div className="font-semibold">
        {parseLocalDate(tooltip.date).toLocaleDateString("en-US", {
          weekday: "short",
          day: "numeric",
          month: "short",
          year: "numeric",
        })}
      </div>
      <div style={{ color: "hsl(var(--primary))" }}>
        {tooltip.count} {tooltip.count === 1 ? "beer" : "beers"}
      </div>
    </div>
  );
}
