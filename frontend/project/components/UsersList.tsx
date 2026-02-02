"use client"
import React, { useEffect, useState, useRef } from 'react'
import Image from 'next/image'

type DateRange = { start?: string; end?: string }
type UsersListProps = {
  title: 'Givers' | 'Recipients' | string
  users: string[]
  range: DateRange
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

  // Fetch real names and avatars for each user
  useEffect(() => {
    let cancelled = false
    async function fetchNamesAvatars() {
      const namesOut = {}
      const avatarsOut = {}
      const concurrency = 5
      let idx = 0
      async function worker() {
        while (idx < users.length && !cancelled) {
          const i = idx++
          const u = users[i]
          try {
            const resp = await fetch(`/api/proxy/user?user=${encodeURIComponent(u)}`)
            if (!resp.ok) {
              namesOut[u] = u
              avatarsOut[u] = null
              continue
            }
            const j = await resp.json()
            namesOut[u] = j.real_name || u
            avatarsOut[u] = j.profile_image || null
          } catch (err) {
            console.error(err);
            namesOut[u] = u
            avatarsOut[u] = null
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
