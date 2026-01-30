"use client"

import { useEffect, useState } from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme } from './ThemeProvider'

export default function ThemeToggle() {
  const { toggleTheme } = useTheme()
  const [isDark, setIsDark] = useState(false)

  useEffect(() => {
    if (typeof document !== 'undefined') {
      setIsDark(document.documentElement.classList.contains('dark'))
    }
  }, [])

  function onClick() {
    toggleTheme()
    setIsDark((v) => !v)
  }

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      className="ml-4 inline-flex items-center gap-2 rounded-md px-3 py-1 text-sm font-medium bordered cursor-pointer"
      style={{ backgroundColor: 'hsl(var(--card))', color: 'hsl(var(--card-foreground))' }}
    >
      {isDark ? <Sun size={16} /> : <Moon size={16} />}
      <span className="sr-only">Toggle theme</span>
    </button>
  )
}
