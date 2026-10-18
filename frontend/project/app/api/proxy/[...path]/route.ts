import { NextRequest, NextResponse } from 'next/server'
import { backendApiUrl, backendAuthHeaders } from '@/lib/server/backend'

type ProxyParams = { path?: string[] | string }

// Request headers passed on to the backend. Authorization is deliberately not among them:
// the proxy exists so the browser never holds a token, so it always uses API_TOKEN.
const FORWARDED_REQUEST_HEADERS = ['accept', 'content-type', 'if-none-match', 'if-modified-since']

// Response headers passed back to the browser, so backend caching (Cache-Control/ETag) works.
// Content-Encoding and Content-Length are not forwarded: fetch() has already decoded the body.
const FORWARDED_RESPONSE_HEADERS = ['content-type', 'cache-control', 'etag', 'last-modified', 'expires', 'vary']

// Statuses that must not carry a body.
const NULL_BODY_STATUSES = new Set([204, 205, 304])

async function proxy(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  const p = await ctx.params
  const pathArr = Array.isArray(p?.path) ? p.path : (p?.path ? [p.path] : [])
  const url = backendApiUrl(pathArr.join('/'), request.nextUrl?.search || '')

  const headers = backendAuthHeaders()
  for (const name of FORWARDED_REQUEST_HEADERS) {
    const value = request.headers.get(name)
    if (value) headers.set(name, value)
  }

  let body: ArrayBuffer | undefined
  if (!['GET', 'HEAD'].includes(request.method)) {
    body = await request.arrayBuffer()
  }

  try {
    const resp = await fetch(url, {
      method: request.method,
      headers,
      body,
      redirect: 'manual',
      cache: 'no-store',
    })

    const outHeaders = new Headers()
    for (const name of FORWARDED_RESPONSE_HEADERS) {
      const value = resp.headers.get(name)
      if (value) outHeaders.set(name, value)
    }
    if (!outHeaders.has('content-type')) outHeaders.set('content-type', 'text/plain')

    // Stream the body through instead of buffering it.
    const outBody = NULL_BODY_STATUSES.has(resp.status) || request.method === 'HEAD' ? null : resp.body
    return new NextResponse(outBody, { status: resp.status, headers: outHeaders })
  } catch (err) {
    return NextResponse.json({ error: (err as Error).message }, { status: 502 })
  }
}

export async function GET(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}

export async function HEAD(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}

export async function POST(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}

export async function PUT(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}

export async function PATCH(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}

export async function DELETE(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}

export async function OPTIONS(request: NextRequest, ctx: { params: Promise<ProxyParams> }) {
  return proxy(request, ctx)
}
