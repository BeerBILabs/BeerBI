"use client"

import { useEffect, useState } from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme } from './ThemeProvider'

export default function ThemeToggle() {
  const { toggleTheme, getTheme } = useTheme()
  const [isDark, setIsDark] = useState(false)
  const [showTooltip, setShowTooltip] = useState(false)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
    setIsDark(getTheme() === 'dark')
  }, [getTheme])

  function onClick() {
    toggleTheme()
    setIsDark((v) => !v)
  }

  // Don't render toggle until mounted to avoid hydration mismatch
  if (!mounted) {
    return (
      <div className="ml-4 w-[72px] h-[32px]" aria-hidden="true" />
    )
  }

  const tooltipText = isDark ? 'Switch to light mode' : 'Switch to dark mode'

  return (
    <div className="relative ml-4">
      <button
        type="button"
        onClick={onClick}
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        onFocus={() => setShowTooltip(true)}
        onBlur={() => setShowTooltip(false)}
        aria-label={tooltipText}
        className="inline-flex items-center justify-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium cursor-pointer border transition-colors duration-200"
        style={{
          backgroundColor: 'hsl(var(--card))',
          color: 'hsl(var(--card-foreground))',
          borderColor: 'hsl(var(--border))',
        }}
      >
        <span
          className="relative w-4 h-4 transition-transform duration-200"
          style={{ transform: isDark ? 'rotate(0deg)' : 'rotate(180deg)' }}
        >
          {isDark ? (
            <Sun size={16} className="absolute inset-0" />
          ) : (
            <Moon size={16} className="absolute inset-0" />
          )}
        </span>
      </button>

      {/* Tooltip */}
      {showTooltip && (
        <div
          role="tooltip"
          className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-2 py-1 text-xs rounded whitespace-nowrap pointer-events-none z-50 transition-opacity duration-150"
          style={{
            backgroundColor: 'hsl(var(--popover))',
            color: 'hsl(var(--popover-foreground))',
            border: '1px solid hsl(var(--border))',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
          }}
        >
          {tooltipText}
          {/* Tooltip arrow */}
          <div
            className="absolute left-1/2 -translate-x-1/2 top-full w-0 h-0"
            style={{
              borderLeft: '4px solid transparent',
              borderRight: '4px solid transparent',
              borderTop: '4px solid hsl(var(--border))',
            }}
          />
        </div>
      )}
    </div>
  )
}
