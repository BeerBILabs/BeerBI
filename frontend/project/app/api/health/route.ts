import { NextResponse } from 'next/server'
import { backendBase } from '@/lib/server/backend'

export async function GET(): Promise<NextResponse> {
  const backend = backendBase()
  try {
    const resp = await fetch(`${backend}/healthz`, { method: 'GET', cache: 'no-store' })
    if (!resp.ok) {
      return NextResponse.json({ status: 'unhealthy', backend: resp.status }, { status: 502 })
    }
    return NextResponse.json({ status: 'ok', backend }, { status: 200 })
  } catch (err) {
    const message = (err as Error)?.message ?? 'unknown error'
    return NextResponse.json({ status: 'unreachable', error: message }, { status: 502 })
  }
}
