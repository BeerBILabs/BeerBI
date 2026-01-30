"use client"

import { useEffect, useState } from 'react'
import type { ReactElement } from 'react'
import UsersList from './UsersList'
import DateRangePicker from './DateRangePicker'

type DateRange = { start: string; end: string }

const quickRanges: Array<{ label: string; get: () => DateRange }> = [
  {
    label: 'Today',
    get: () => {
      const d = new Date();
      const s = d.toISOString().slice(0, 10);
      return { start: s, end: s };
    }
  },
  {
    label: 'Yesterday',
    get: () => {
      const d = new Date();
      d.setDate(d.getDate() - 1);
      const s = d.toISOString().slice(0, 10);
      return { start: s, end: s };
    }
  },
  {
    label: 'Last Week',
    get: () => {
      const d = new Date();
      const end = d.toISOString().slice(0, 10);
      d.setDate(d.getDate() - 6);
      const start = d.toISOString().slice(0, 10);
      return { start, end };
    }
  },
  {
    label: 'Last Month',
    get: () => {
      const d = new Date();
      const end = d.toISOString().slice(0, 10);
      d.setMonth(d.getMonth() - 1);
      const start = d.toISOString().slice(0, 10);
      return { start, end };
    }
  },
  {
    label: 'Last 3 Months',
    get: () => {
      const d = new Date();
      const end = d.toISOString().slice(0, 10);
      d.setMonth(d.getMonth() - 3);
      const start = d.toISOString().slice(0, 10);
      return { start, end };
    }
  },
  {
    label: 'This Year',
    get: () => {
      const now = new Date();
      const start = new Date(now.getFullYear(), 0, 1).toISOString().slice(0, 10);
      const end = now.toISOString().slice(0, 10);
      return { start, end };
    }
  },
  {
    label: 'Last Year',
    get: () => {
      const now = new Date();
      const start = new Date(now.getFullYear() - 1, 0, 1).toISOString().slice(0, 10);
      const end = new Date(now.getFullYear() - 1, 11, 31).toISOString().slice(0, 10);
      return { start, end };
    }
  }
];

export default function UsersPage(): ReactElement {
  // Default to current year
  const now = new Date()
  const yearStart = new Date(now.getFullYear(), 0, 1)
  const [range, setRange] = useState<DateRange>({
    start: yearStart.toISOString().slice(0, 10),
    end: now.toISOString().slice(0, 10)
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
      <h1 className="text-3xl font-extrabold mb-2 text-indigo-700 tracking-tight flex items-center gap-2">
        <span>🍺</span> BeerBot Leaderboard
      </h1>
      <p className="mb-6 text-gray-500 text-base">See who’s giving and receiving the most <span className="font-semibold text-yellow-500">beers</span>! Select a date range or use a quick filter.</p>

      <div className="mb-6 flex flex-col gap-3 items-center">
        <div className="flex flex-wrap gap-2 mb-2 justify-center">
          {quickRanges.map(q => (
            <button
              key={q.label}
              className="px-3 py-1 rounded bg-indigo-100 hover:bg-indigo-300 text-indigo-900 text-sm font-medium border border-indigo-200 transition-colors duration-150 shadow-sm"
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
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8 animate-fadein">
        <UsersList title="Givers" users={givers} range={range} />
        <UsersList title="Recipients" users={recipients} range={range} />
      </div>
    </section>
  )
}
