// Server-only helpers for talking to the Go backend. It reads API_TOKEN, which must not reach the
// browser bundle: `server-only` makes the build fail if a client component imports this module.
import 'server-only'

const DEFAULT_BACKEND_BASE = 'http://localhost:8080'

/** Backend base URL from BACKEND_BASE (runtime, server-side only), without a trailing slash. */
export function backendBase(): string {
  return (process.env.BACKEND_BASE || DEFAULT_BACKEND_BASE).replace(/\/+$/, '')
}

/** Headers every backend API call carries: the bearer token from API_TOKEN, if set. */
export function backendAuthHeaders(): Headers {
  const headers = new Headers()
  const token = process.env.API_TOKEN
  if (token) headers.set('Authorization', `Bearer ${token}`)
  return headers
}

/** URL of a backend API endpoint, e.g. backendApiUrl('stats/top', '?limit=10'). */
export function backendApiUrl(path: string, search = ''): string {
  return `${backendBase()}/api/${path}${search}`
}
