"use client"
import { useId, useState } from 'react'
import type { ReactElement } from 'react'

type DateRangePickerProps = {
  start?: string | null
  end?: string | null
  onChange?: (start: string, end: string) => void
}

// A native date input reports a complete date as YYYY-MM-DD and an incomplete or cleared one as ''.
// While the year is being typed, Chrome reports intermediate years (0002, 0020, 0202, 2026), so a
// date only counts once the year has four digits and does not start with 0.
const COMPLETE_DATE = /^[1-9]\d{3}-\d{2}-\d{2}$/

type DateFieldProps = {
  id: string
  name: string
  label: string
  value: string
  min?: string
  max?: string
  /** A complete date was entered. Called for every such change while typing. */
  onEnter: (value: string) => void
  /** The field was left (blur) or confirmed (Enter) with a complete date that was not applied yet. */
  onSettle: (value: string) => void
}

function DateField({ id, name, label, value, min, max, onEnter, onSettle }: DateFieldProps): ReactElement {
  // Keep what the user is typing (a half-edited date reads as '') without pushing it up
  // or letting React reset the segments; follow the prop whenever it changes.
  const [draft, setDraft] = useState(value)
  const [seen, setSeen] = useState(value)
  if (value !== seen) {
    setSeen(value)
    setDraft(value)
  }

  function settle() {
    if (draft === value) return
    if (COMPLETE_DATE.test(draft)) onSettle(draft)
    else setDraft(value) // emptied or half-typed: back to the last applied date
  }

  return (
    <div className="flex-1 min-w-[130px]">
      <label htmlFor={id} className="sr-only">
        {label}
      </label>
      <input
        id={id}
        name={name}
        type="date"
        value={draft}
        min={min}
        max={max}
        onChange={(e) => {
          const next = e.target.value
          setDraft(next)
          if (COMPLETE_DATE.test(next)) onEnter(next)
        }}
        onBlur={settle}
        onKeyDown={(e) => {
          if (e.key === 'Enter') settle()
        }}
        className="rounded border px-2 py-1 cursor-pointer w-full min-w-0 text-sm"
        style={{
          backgroundColor: 'hsl(var(--card))',
          color: 'hsl(var(--card-foreground))',
          borderColor: 'hsl(var(--border))',
          transition: 'color 250ms ease-in-out, background-color 250ms ease-in-out, border-color 250ms ease-in-out',
        }}
      />
    </div>
  )
}

export default function DateRangePicker({ start, end, onChange }: DateRangePickerProps): ReactElement {
  const id = useId()
  const startValue = start ?? ''
  const endValue = end ?? ''

  // Inline style ensures consistent 250ms transition for all color properties
  const inputStyle = {
    backgroundColor: 'hsl(var(--card))',
    color: 'hsl(var(--card-foreground))',
    borderColor: 'hsl(var(--border))',
    transition: 'color 250ms ease-in-out, background-color 250ms ease-in-out, border-color 250ms ease-in-out',
  };

  // Entering a date emits it right away when it fits the other bound (the min/max attributes
  // keep the picker popup inside the range). A typed date past the other bound is only applied
  // when the field is left or confirmed, and then moves that bound along, so typing a year
  // never drags the other date around keystroke by keystroke.
  return (
    <div
      className="flex flex-wrap items-center gap-2 sm:gap-3 rounded-lg px-3 py-2 shadow-sm border w-full max-w-full"
      style={inputStyle}
      role="group"
      aria-label="Date range"
    >
      <DateField
        id={`${id}-from`}
        name="date-from"
        label="Start date"
        value={startValue}
        max={endValue || undefined}
        onEnter={(s) => {
          if (!endValue || s <= endValue) onChange?.(s, endValue || s)
        }}
        onSettle={(s) => onChange?.(s, endValue && endValue >= s ? endValue : s)}
      />
      <span className="font-bold" style={{ color: 'hsl(var(--muted-foreground))' }} aria-hidden="true">to</span>
      <DateField
        id={`${id}-to`}
        name="date-to"
        label="End date"
        value={endValue}
        min={startValue || undefined}
        onEnter={(e) => {
          if (!startValue || e >= startValue) onChange?.(startValue || e, e)
        }}
        onSettle={(e) => onChange?.(startValue && startValue <= e ? startValue : e, e)}
      />
    </div>
  )
}
