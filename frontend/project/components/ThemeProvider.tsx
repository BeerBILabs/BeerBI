"use client"

import { useEffect } from 'react'

type Theme = 'light' | 'dark'

const THEME_COOKIE = 'theme'
const ONE_YEAR_SECONDS = 60 * 60 * 24 * 365

function readThemeCookie(): string | undefined {
  return document.cookie
    .split('; ')
    .find(row => row.startsWith(`${THEME_COOKIE}=`))
    ?.split('=')[1]
}

// Client-side only: no server action, so there is no extra POST on the first visit.
// Name and format are what app/layout.tsx and its blocking ThemeScript read.
function writeThemeCookie(theme: Theme) {
  document.cookie = `${THEME_COOKIE}=${theme}; path=/; max-age=${ONE_YEAR_SECONDS}; SameSite=Lax`
}

export function ThemeProvider() {
  useEffect(() => {
    // Without a theme cookie the page is light: make sure of it and remember the choice.
    if (!readThemeCookie()) {
      document.documentElement.classList.remove('dark')
      writeThemeCookie('light')
    }
  }, [])

  return null
}

export function useTheme() {
  function setTheme(theme: Theme) {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    writeThemeCookie(theme)
  }

  function toggleTheme() {
    const isDark = document.documentElement.classList.contains('dark')
    setTheme(isDark ? 'light' : 'dark')
  }

  function getTheme(): Theme {
    if (typeof document === 'undefined') return 'light'
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  }

  return { setTheme, toggleTheme, getTheme }
}
