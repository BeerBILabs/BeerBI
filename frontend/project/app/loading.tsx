import type { ReactElement } from 'react'

// Shown while a route segment loads during client-side navigation.
export default function Loading(): ReactElement {
  return (
    <main className="flex-1 flex items-center justify-center px-8 py-20" aria-busy="true">
      <p className="text-base animate-pulse" style={{ color: 'hsl(var(--muted-foreground))' }}>
        Loading…
      </p>
    </main>
  )
}
