"use client"

import { useEffect, useState } from 'react'
import { setThemeCookie } from '@/app/actions/theme'

export function ThemeProvider() {
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    // Read cookie on mount (cookie already applied server-side, but sync state)
    const theme = document.cookie
      .split('; ')
      .find(row => row.startsWith('theme='))
      ?.split('=')[1] as 'light' | 'dark' | undefined

    // If no cookie exists, default to light and set cookie
    if (!theme) {
      document.documentElement.classList.remove('dark')
      setThemeCookie('light')
    }

    setMounted(true)
  }, [])

  return mounted ? null : null
}

export function useTheme() {
  function setTheme(theme: 'light' | 'dark') {
    const root = document.documentElement
    root.classList.toggle('dark', theme === 'dark')
    // Fire and forget - don't await to avoid blocking UI
    setThemeCookie(theme)
    // Also set client-side cookie for immediate reads (server action may have latency)
    document.cookie = `theme=${theme}; path=/; max-age=31536000; SameSite=Lax`
  }

  function toggleTheme() {
    const isDark = document.documentElement.classList.contains('dark')
    setTheme(isDark ? 'light' : 'dark')
  }

  function getTheme(): 'light' | 'dark' {
    if (typeof document === 'undefined') return 'light'
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  }

  return { setTheme, toggleTheme, getTheme }
}
