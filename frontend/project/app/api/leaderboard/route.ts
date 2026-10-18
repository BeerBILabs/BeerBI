import { NextRequest, NextResponse } from 'next/server'
import { backendApiUrl, backendAuthHeaders } from '@/lib/server/backend'
import type { LeaderboardResponse, TopUsersResult } from '@/lib/statsTypes'

// GET /api/leaderboard?start=&end=[&prev_start=&prev_end=][&limit=]
//
// One browser request for a leaderboard: the top givers/recipients of a period and, optionally, of
// a comparison period (previous quarter, for the rank-change arrows). Both backend calls run in
// parallel on the server, so the browser no longer waits for one before starting the other.
//
// Caching: the backend marks each /stats/top answer with Cache-Control (long max-age for closed
// periods, short for the running one). The combined response may be cached by the browser for the
// shorter of the two max-age values, always `private`; if either part gives no max-age (or says
// no-store/no-cache), or the comparison period failed, it is not cached. The combined body has no
// ETag of its own (conditional requests are not forwarded).

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/
const DEFAULT_LIMIT = 100
const MAX_LIMIT = 100

function badRequest(message: string) {
  return NextResponse.json({ error: message }, { status: 400 })
}

type TopFetch = { data: TopUsersResult; maxAge: number | null }

/** Seconds a response may be cached for according to its Cache-Control, or null if not at all. */
function cacheableSeconds(cacheControl: string | null): number | null {
  if (!cacheControl) return null
  const directives = cacheControl.toLowerCase().split(',').map((d) => d.trim())
  if (directives.some((d) => d === 'no-store' || d === 'no-cache')) return null
  const maxAge = directives.find((d) => d.startsWith('max-age='))
  const seconds = maxAge ? Number(maxAge.slice('max-age='.length)) : NaN
  return Number.isInteger(seconds) && seconds > 0 ? seconds : null
}

async function fetchTop(start: string, end: string, limit: number): Promise<TopFetch> {
  const params = new URLSearchParams({ start, end, limit: String(limit) })
  const resp = await fetch(backendApiUrl('stats/top', `?${params}`), {
    headers: backendAuthHeaders(),
    cache: 'no-store',
  })
  if (!resp.ok) throw new Error(`backend returned ${resp.status}`)
  const data = (await resp.json()) as Partial<TopUsersResult> | null
  return {
    data: { givers: data?.givers ?? [], recipients: data?.recipients ?? [] },
    maxAge: cacheableSeconds(resp.headers.get('cache-control')),
  }
}

export async function GET(request: NextRequest): Promise<NextResponse> {
  const q = request.nextUrl.searchParams
  const start = q.get('start') ?? ''
  const end = q.get('end') ?? ''
  const prevStart = q.get('prev_start')
  const prevEnd = q.get('prev_end')

  if (!DATE_RE.test(start) || !DATE_RE.test(end)) return badRequest('start and end must be YYYY-MM-DD')
  if ((prevStart === null) !== (prevEnd === null)) return badRequest('prev_start and prev_end go together')
  if (prevStart !== null && (!DATE_RE.test(prevStart) || !DATE_RE.test(prevEnd ?? ''))) {
    return badRequest('prev_start and prev_end must be YYYY-MM-DD')
  }

  const limitParam = q.get('limit')
  const limit = limitParam === null ? DEFAULT_LIMIT : Number(limitParam)
  if (!Number.isInteger(limit) || limit < 1 || limit > MAX_LIMIT) {
    return badRequest(`limit must be an integer between 1 and ${MAX_LIMIT}`)
  }

  const [current, previous] = await Promise.allSettled([
    fetchTop(start, end, limit),
    prevStart !== null && prevEnd !== null ? fetchTop(prevStart, prevEnd, limit) : Promise.resolve(null),
  ])

  if (current.status === 'rejected') {
    return NextResponse.json({ error: `Failed to fetch top users: ${(current.reason as Error).message}` }, { status: 502 })
  }

  // The comparison period only drives the rank-change arrows: show the ranking without them on failure.
  const prev = previous.status === 'fulfilled' ? previous.value : null
  const body: LeaderboardResponse = { current: current.value.data, previous: prev?.data ?? null }

  const parts = [current.value.maxAge]
  if (prevStart !== null) parts.push(prev ? prev.maxAge : null)
  const maxAge = parts.every((s): s is number => s !== null) ? Math.min(...parts) : null
  return NextResponse.json(body, {
    headers: { 'Cache-Control': maxAge === null ? 'private, no-store' : `private, max-age=${maxAge}` },
  })
}
