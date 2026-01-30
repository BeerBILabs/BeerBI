"use client"

import { useEffect, useState } from 'react'

function getSystemPrefersDark(): boolean {
  if (typeof window === 'undefined') return false
  return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
}

export function ThemeProvider() {
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    const stored = typeof window !== 'undefined' ? localStorage.getItem('theme') : null
    const shouldDark = stored ? stored === 'dark' : getSystemPrefersDark()
    const root = document.documentElement
    root.classList.toggle('dark', shouldDark)
    setMounted(true)
  }, [])

  // non-visual helper; ensures class applied on mount
  return mounted ? null : null
}

export function useTheme() {
  function setTheme(theme: 'light' | 'dark') {
    const root = document.documentElement
    root.classList.toggle('dark', theme === 'dark')
    localStorage.setItem('theme', theme)
  }
  function toggleTheme() {
    const isDark = document.documentElement.classList.contains('dark')
    setTheme(isDark ? 'light' : 'dark')
  }
  return { setTheme, toggleTheme }
}
