"use client"

import type { ReactElement } from 'react'

// Error boundary for the page segments; the layout (header, theme) stays usable.
export default function RouteError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}): ReactElement {
  return (
    <main className="flex-1 flex flex-col items-center justify-center gap-4 px-8 py-20" role="alert">
      <p className="text-lg" style={{ color: 'hsl(var(--destructive))' }}>
        Something went wrong{error.digest ? ` (${error.digest})` : ''}.
      </p>
      <button
        type="button"
        onClick={reset}
        className="px-4 py-2 rounded-md text-sm font-semibold transition-colors cursor-pointer"
        style={{ backgroundColor: 'hsl(var(--primary))', color: 'hsl(var(--primary-foreground))' }}
      >
        Try again
      </button>
    </main>
  )
}
