"use client"

import { useState } from 'react'
import type { ReactElement } from 'react'
import { Leaderboard } from './Leaderboard'
import DateRangePicker from './DateRangePicker'
import { formatLocalDate } from '@/lib/dateUtils'
import { useHydrated } from '@/lib/useHydrated'

type DateRange = { start: string; end: string }

const quickRanges: Array<{ label: string; get: () => DateRange }> = [
  {
    label: 'Today',
    get: () => {
      const d = new Date();
      const s = formatLocalDate(d);
      return { start: s, end: s };
    }
  },
  {
    label: 'Yesterday',
    get: () => {
      const d = new Date();
      d.setDate(d.getDate() - 1);
      const s = formatLocalDate(d);
      return { start: s, end: s };
    }
  },
  {
    label: 'Last Week',
    get: () => {
      const d = new Date();
      const end = formatLocalDate(d);
      d.setDate(d.getDate() - 6);
      const start = formatLocalDate(d);
      return { start, end };
    }
  },
  {
    label: 'Last Month',
    get: () => {
      const d = new Date();
      const end = formatLocalDate(d);
      d.setMonth(d.getMonth() - 1);
      const start = formatLocalDate(d);
      return { start, end };
    }
  },
  {
    label: 'Last 3 Months',
    get: () => {
      const d = new Date();
      const end = formatLocalDate(d);
      d.setMonth(d.getMonth() - 3);
      const start = formatLocalDate(d);
      return { start, end };
    }
  },
];

/** Default range: the current year up to today. */
function thisYearSoFar(): DateRange {
  const now = new Date()
  return { start: formatLocalDate(new Date(now.getFullYear(), 0, 1)), end: formatLocalDate(now) }
}

export default function UsersPage(): ReactElement {
  // The default range comes from the client's clock after hydration; the page HTML is built once
  // at build time and must not contain a date (see useHydrated).
  const hydrated = useHydrated()
  const [picked, setRange] = useState<DateRange | null>(null)
  const range = picked ?? (hydrated ? thisYearSoFar() : null)

  return (
    <section>
      <h1
        className="text-3xl font-extrabold mb-2 tracking-tight"
        style={{ color: 'hsl(var(--primary))' }}
      >
        BeerBot Leaderboard
      </h1>
      <p className="mb-6 text-base" style={{ color: 'hsl(var(--muted-foreground))' }}>
        See who's giving and receiving the most{' '}
        <span className="font-semibold" style={{ color: 'hsl(var(--primary))' }}>beers</span>!
        Select a date range or use a quick filter.
      </p>

      <div className="mb-6 flex flex-col gap-3 items-center">
        <div className="flex flex-wrap gap-2 mb-2 justify-center">
          {quickRanges.map(q => (
            <button
              key={q.label}
              className="px-3 py-1 rounded text-sm font-medium border transition-colors duration-150 shadow-sm cursor-pointer"
              style={{
                backgroundColor: 'hsl(var(--secondary))',
                color: 'hsl(var(--secondary-foreground))',
                borderColor: 'hsl(var(--border))',
              }}
              onClick={() => setRange(q.get())}
              type="button"
            >
              {q.label}
            </button>
          ))}
        </div>
        <div className="flex justify-center">
          <DateRangePicker
            start={range?.start ?? null}
            end={range?.end ?? null}
            onChange={(s, e) => setRange({ start: s, end: e })}
          />
        </div>
      </div>

      <div className="animate-fadein">
        {range ? (
          <Leaderboard
            dateRange={range}
            showRankChange={false}
            maxHeight=""
          />
        ) : (
          <div className="min-h-[400px]" aria-busy="true" />
        )}
      </div>
    </section>
  )
}
