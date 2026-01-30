import Link from 'next/link'
import Logo from '../components/Logo'

export default function NotFound() {
  return (
    <div className="min-h-[60vh] flex flex-col items-center justify-center gap-6 text-center">
      <Logo className="h-12 w-auto" />
      <h1 className="text-2xl font-semibold">Damn, we got nothing for this... :(</h1>
      <Link
        href="/"
        className="inline-flex items-center justify-center rounded-md px-4 py-2 font-medium bordered cursor-pointer"
        style={{ backgroundColor: 'hsl(var(--card))', color: 'hsl(var(--card-foreground))' }}
      >
        Return Home
      </Link>
    </div>
  )
}
