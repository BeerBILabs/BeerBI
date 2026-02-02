import './globals.css'
import type { Metadata } from 'next'
import type { ReactNode } from 'react'
import { Inter } from 'next/font/google'
import { cookies } from 'next/headers'
import { ThemeProvider } from '../components/ThemeProvider'
import ThemeToggle from '../components/ThemeToggle'
import Logo from '../components/Logo'
import Link from 'next/link'

const inter = Inter({ subsets: ['latin'], weight: ['400', '600', '700'] })

export const metadata: Metadata = {
  title: 'BeerBot Frontend',
  description: 'List givers and recipients and stats',
  icons: { icon: '/favicon.svg' },
}

function ThemeScript() {
  // Blocking script that runs before React hydration to prevent FOUC
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

export default async function RootLayout({ children }: { children: ReactNode }) {
  const cookieStore = await cookies()
  const theme = cookieStore.get('theme')?.value || 'light'

  return (
    <html lang="en" className={theme === 'dark' ? 'dark' : ''} suppressHydrationWarning>
      <head>
        <ThemeScript />
      </head>
      <body className={`${inter.className} min-h-screen surface flex flex-col`}>
        <ThemeProvider />
        <header className="w-full surface-inverse border-b bordered shadow-sm sticky top-0 z-30">
          <div className="max-w-5xl mx-auto flex items-center gap-4 px-8 py-4">
            <Logo className="h-10 w-auto" />
            <span className="ml-auto text-base font-medium tracking-tight"><Link href="/">Stats &amp; Leaderboard</Link></span>
            <ThemeToggle />
          </div>
        </header>
        <main className="flex-1 flex flex-col items-center justify-center px-2">
          <div className="w-full max-w-3xl card rounded-2xl shadow-lg p-8 mt-10 mb-10 border bordered">
            {children}
          </div>
        </main>
        <footer className="w-full text-center text-xs py-6 border-t bordered surface-inverse">
          <span>Made with <span className="text-[#d4a84b] font-bold">&#9829;</span> by BeerBot Team</span>
        </footer>
      </body>
    </html>
  )
}
