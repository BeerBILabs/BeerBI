"use client"

import { useEffect, useState } from 'react'
import type { ReactElement } from 'react'
import Link from 'next/link'
import UsersList from './UsersList'
import DateRangePicker from './DateRangePicker'

type DateRange = { start: string; end: string }

// Format date as YYYY-MM-DD in local timezone (avoids UTC conversion issues)
function formatLocalDate(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

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
  {
    label: 'This Year',
    get: () => {
      const now = new Date();
      const start = formatLocalDate(new Date(now.getFullYear(), 0, 1));
      const end = formatLocalDate(now);
      return { start, end };
    }
  },
  {
    label: 'Last Year',
    get: () => {
      const now = new Date();
      const start = formatLocalDate(new Date(now.getFullYear() - 1, 0, 1));
      const end = formatLocalDate(new Date(now.getFullYear() - 1, 11, 31));
      return { start, end };
    }
  }
];

export default function UsersPage(): ReactElement {
  // Default to current year
  const now = new Date()
  const yearStart = new Date(now.getFullYear(), 0, 1)
  const [range, setRange] = useState<DateRange>({
    start: formatLocalDate(yearStart),
    end: formatLocalDate(now)
  })
  const [givers, setGivers] = useState<string[]>([])
  const [recipients, setRecipients] = useState<string[]>([])
  // The proxy will inject an API token from server-side environment variables.
  useEffect(() => {
    async function load() {
      const gResp = await fetch('/api/proxy/givers')
      const rResp = await fetch('/api/proxy/recipients')
      if (gResp.ok) setGivers(await gResp.json())
      if (rResp.ok) setRecipients(await rResp.json())
    }
    load()
  }, [])

  return (
    <section>
      <h1
        className="text-3xl font-extrabold mb-2 tracking-tight flex items-center gap-2"
        style={{ color: 'hsl(var(--primary))' }}
      >
        <span>🍺</span> BeerBot Leaderboard
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
              className="px-3 py-1 rounded text-sm font-medium border transition-colors duration-150 shadow-sm"
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
            start={range.start}
            end={range.end}
            onChange={(s, e) => setRange({ start: s, end: e })}
          />
        </div>
        <div className="flex justify-center mt-4">
          <Link
            href="/rankings"
            className="px-4 py-2 rounded-md text-sm font-semibold transition-colors"
            style={{
              backgroundColor: 'hsl(var(--primary))',
              color: 'hsl(var(--primary-foreground))',
            }}
          >
            Explore Quarterly Rankings
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8 animate-fadein">
        <UsersList title="Givers" users={givers} range={range} />
        <UsersList title="Recipients" users={recipients} range={range} />
      </div>
    </section>
  )
}
