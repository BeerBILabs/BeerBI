import './globals.css'
import type { Metadata } from 'next'
import type { ReactNode } from 'react'
import localFont from 'next/font/local'
import { ThemeProvider } from '../components/ThemeProvider'
import ThemeToggle from '../components/ThemeToggle'
import Logo from '../components/Logo'
import Link from 'next/link'

// Self-hosted Inter (variable weight, Latin subset) from @fontsource-variable/inter: no font
// download from Google during `next build`, which made CI flaky, and no third-party request at runtime.
const inter = localFont({
  src: '../node_modules/@fontsource-variable/inter/files/inter-latin-wght-normal.woff2',
  weight: '100 900',
  style: 'normal',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'BeerBot Frontend',
  description: 'List givers and recipients and stats',
  icons: { icon: '/favicon.svg' },
}

function ThemeScript() {
  // Blocking script that runs before React hydration to prevent FOUC.
  // It is the only place the theme cookie is read for the initial paint: reading it on the
  // server (cookies()) would make every route dynamic (audit F1).
  return (
    <script
      dangerouslySetInnerHTML={{
        __html: `
          (function() {
            try {
              var theme = document.cookie
                .split('; ')
                .find(function(row) { return row.startsWith('theme='); });
              if (theme) {
                theme = theme.split('=')[1];
                if (theme === 'dark') {
                  document.documentElement.classList.add('dark');
                } else {
                  document.documentElement.classList.remove('dark');
                }
              }
            } catch (e) {}
          })();
        `,
      }}
    />
  )
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    // suppressHydrationWarning: ThemeScript may add the `dark` class before React hydrates.
    <html lang="en" className="h-full" suppressHydrationWarning>
      <head>
        <ThemeScript />
      </head>
      <body className={`${inter.className} h-full min-h-screen surface flex flex-col`}>
        <ThemeProvider />
        <header className="w-full surface-inverse border-b bordered shadow-sm sticky top-0 z-30">
          <div className="max-w-5xl mx-auto flex items-center gap-4 px-8 py-4">
            <Link href="/"><Logo className="h-10 w-auto" /></Link>
            <span className="ml-auto text-base font-medium tracking-tight"><Link href="/">Leaderboard</Link></span>
            <Link href="/analytics" className="text-base font-medium tracking-tight hover:opacity-80 transition-opacity">Analytics</Link>
            <ThemeToggle />
          </div>
        </header>
        <div className="flex-1 surface">
          {children}
        </div>
        <footer className="w-full text-center text-xs py-6 border-t bordered surface-inverse">
          <span>Made with <span className="text-[#d4a84b] font-bold">&#9829;</span> by BeerBot Team</span>
        </footer>
      </body>
    </html>
  )
}
