import { NextResponse } from 'next/server'

export async function GET(): Promise<NextResponse> {
  const backend = process.env.NEXT_PUBLIC_BACKEND_BASE || 'http://localhost:8080'
  try {
    const resp = await fetch(`${backend}/healthz`, { method: 'GET' })
    if (!resp.ok) {
      return NextResponse.json({ status: 'unhealthy', backend: resp.status }, { status: 502 })
    }
    return NextResponse.json({ status: 'ok', backend }, { status: 200 })
  } catch (err) {
    const message = (err as Error)?.message ?? 'unknown error'
    return NextResponse.json({ status: 'unreachable', error: message }, { status: 502 })
  }
}
