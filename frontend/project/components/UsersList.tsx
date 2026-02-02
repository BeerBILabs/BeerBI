"use client"
import React, { useEffect, useState, useRef } from 'react'
import Image from 'next/image'

type DateRange = { start?: string; end?: string }
type UsersListProps = {
  title: 'Givers' | 'Recipients' | string
  users: string[]
  range: DateRange
}

type CachedUserInfo = {
  real_name: string
  profile_image: string | null
  cached_at: number
}

const USER_CACHE_KEY = 'beerbot_user_cache'
const CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000 // 7 days

function getUserCache(): Record<string, CachedUserInfo> {
  if (typeof window === 'undefined') return {}
  try {
    const raw = localStorage.getItem(USER_CACHE_KEY)
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
  }
}

function setUserCache(cache: Record<string, CachedUserInfo>): void {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(USER_CACHE_KEY, JSON.stringify(cache))
  } catch {
    // localStorage might be full or disabled
  }
}

function getCachedUser(userId: string): CachedUserInfo | null {
  const cache = getUserCache()
  const entry = cache[userId]
  if (!entry) return null
  // Check if cache is still valid
  if (Date.now() - entry.cached_at > CACHE_TTL_MS) return null
  return entry
}

function setCachedUser(userId: string, realName: string, profileImage: string | null): void {
  const cache = getUserCache()
  cache[userId] = {
    real_name: realName,
    profile_image: profileImage,
    cached_at: Date.now()
  }
  setUserCache(cache)
}

export default function UsersList({ title, users, range }: UsersListProps) {
  const [stats, setStats] = useState<Record<string, number>>({})
  const [names, setNames] = useState<Record<string, string>>({})
  const [avatars, setAvatars] = useState<Record<string, string | null>>({})
  const lastArgsRef = useRef<string | null>(null)
  const mounted = useRef<boolean>(true)

  useEffect(() => {
    mounted.current = true
    return () => { mounted.current = false }
  }, [])

  // Fetch stats for each user
  useEffect(() => {
    const usersKey = JSON.stringify(users || [])
    const rangeKey = `${range?.start ?? ''}_${range?.end ?? ''}`
    const argsKey = `${title}|${usersKey}|${rangeKey}`
    if (lastArgsRef.current === argsKey) return
    lastArgsRef.current = argsKey

    let cancelled = false
    const timer = setTimeout(() => {
      async function loadStats() {
        const out: Record<string, number> = {}
        const path = title === 'Givers' ? '/api/proxy/given' : '/api/proxy/received'
        const params = new URLSearchParams()
        if (range?.start && range?.end) {
          if (range.start === range.end) {
            params.set('day', range.start)
          } else {
            params.set('start', range.start)
            params.set('end', range.end)
          }
        }
        const concurrency = 5
        let idx = 0
        async function worker(): Promise<void> {
          while (idx < users.length && !cancelled) {
            const i = idx++
            const u = users[i]
            const q = new URLSearchParams()
            q.set('user', u)
            const day = params.get('day')
            if (day) q.set('day', day)
            const startV = params.get('start')
            if (startV) q.set('start', startV)
            const endV = params.get('end')
            if (endV) q.set('end', endV)
            try {
              const resp = await fetch(`${path}?${q.toString()}`)
              if (!resp.ok) {
                out[u] = 0
                continue
              }
              const j = await resp.json()
              out[u] = title === 'Givers' ? j.given : j.received
            } catch (err) {
              console.error(err);
              out[u] = 0
            }
          }
        }
        const workers: Promise<void>[] = []
        for (let i = 0; i < concurrency; i++) workers.push(worker())
        await Promise.all(workers)
        if (cancelled || !mounted.current) return
        const prev = JSON.stringify(stats)
        const next = JSON.stringify(out)
        if (prev !== next && mounted.current) setStats(out)
      }
      if (users && users.length) loadStats()
      else setStats({})
    }, 200)
    return () => { cancelled = true; clearTimeout(timer) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [users, range.start, range.end, title])

  // Fetch real names and avatars for each user (with localStorage caching)
  useEffect(() => {
    let cancelled = false
    async function fetchNamesAvatars() {
      const namesOut: Record<string, string> = {}
      const avatarsOut: Record<string, string | null> = {}
      const concurrency = 5
      let idx = 0
      async function worker() {
        while (idx < users.length && !cancelled) {
          const i = idx++
          const u = users[i]
          
          // Check cache first
          const cached = getCachedUser(u)
          if (cached) {
            namesOut[u] = cached.real_name
            avatarsOut[u] = cached.profile_image
            continue
          }
          
          try {
            const resp = await fetch(`/api/proxy/user?user=${encodeURIComponent(u)}`)
            if (!resp.ok) {
              // API failed - try to use expired cache as fallback
              const expiredCache = getUserCache()[u]
              if (expiredCache) {
                namesOut[u] = expiredCache.real_name
                avatarsOut[u] = expiredCache.profile_image
              } else {
                namesOut[u] = u
                avatarsOut[u] = null
              }
              continue
            }
            const j = await resp.json()
            const realName = j.real_name || u
            const profileImage = j.profile_image || null
            namesOut[u] = realName
            avatarsOut[u] = profileImage
            // Cache successful response
            if (j.real_name) {
              setCachedUser(u, realName, profileImage)
            }
          } catch (err) {
            console.error(err);
            // On error, try expired cache as fallback
            const expiredCache = getUserCache()[u]
            if (expiredCache) {
              namesOut[u] = expiredCache.real_name
              avatarsOut[u] = expiredCache.profile_image
            } else {
              namesOut[u] = u
              avatarsOut[u] = null
            }
          }
        }
      }
      const workers: Promise<void>[] = []
      for (let i = 0; i < concurrency; i++) workers.push(worker())
      await Promise.all(workers)
      if (!cancelled && mounted.current) {
        setNames(namesOut)
        setAvatars(avatarsOut)
      }
    }
    if (users && users.length) fetchNamesAvatars()
    else {
      setNames({})
      setAvatars({})
    }
    return () => { cancelled = true }
  }, [users])

  // Sort users by count (desc), show only top 100
  const sorted = (users ?? [])
    .map((u) => ({ user: u, count: stats[u] ?? 0 }))
    .sort((a, b) => (b.count ?? 0) - (a.count ?? 0))
    .slice(0, 100)
  const total = sorted.reduce((sum, u) => sum + (typeof u.count === 'number' ? u.count : 0), 0)

  return (
    <div
      className="p-5 rounded-2xl shadow-lg min-h-[400px] border"
      style={{
        backgroundColor: 'hsl(var(--card))',
        color: 'hsl(var(--card-foreground))',
        borderColor: 'hsl(var(--border))',
      }}
    >
      <h2
        className="text-xl font-bold mb-2 flex items-center gap-2"
        style={{ color: 'hsl(var(--primary))' }}
      >
        {title === 'Givers' ? '🙌' : '🎉'} {title}
      </h2>
      <div className="mb-3 text-sm" style={{ color: 'hsl(var(--muted-foreground))' }}>
        Total <span className="font-bold" style={{ color: 'hsl(var(--primary))' }}>🍺 {total}</span>
      </div>
      {sorted.length === 0 ? (
        <div className="text-sm" style={{ color: 'hsl(var(--muted-foreground))' }}>No data</div>
      ) : (
        <ul className="divide-y" style={{ borderColor: 'hsl(var(--border))' }}>
          {sorted.map(({ user, count }, i) => (
            <li
              key={user}
              className="flex items-center justify-between py-2 group transition rounded-lg px-2 hover:opacity-80"
              style={{
                animation: `fadein 0.3s ${i * 0.01}s both`,
                borderColor: 'hsl(var(--border))',
              }}
            >
              <div className="flex items-center gap-3">
                {avatars[user] ? (
                  <Image
                    src={avatars[user] as string}
                    alt={names[user] || user}
                    width={32}
                    height={32}
                    unoptimized
                    className="w-8 h-8 rounded-full object-cover shadow-sm border"
                    style={{ borderColor: 'hsl(var(--border))' }}
                  />
                ) : (
                  <span
                    className="inline-flex items-center justify-center w-8 h-8 rounded-full font-bold text-lg shadow-sm"
                    style={{
                      backgroundColor: 'hsl(var(--accent))',
                      color: 'hsl(var(--accent-foreground))',
                    }}
                  >
                    {names[user]?.[0]?.toUpperCase() || user[0]}
                  </span>
                )}
                <span className="text-base font-medium" style={{ color: 'hsl(var(--foreground))' }}>
                  {names[user] || user}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-lg">🍺</span>
                <span className="text-base font-bold" style={{ color: 'hsl(var(--primary))' }}>{count}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
