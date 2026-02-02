"use client"
import type { ReactElement } from 'react'
import DatePicker from 'react-datepicker'
import 'react-datepicker/dist/react-datepicker.css'

type DateRangePickerProps = {
  start?: string | null
  end?: string | null
  onChange?: (start: string, end: string) => void
}

export default function DateRangePicker({ start, end, onChange }: DateRangePickerProps): ReactElement {
  // Inline style ensures consistent 250ms transition for all color properties
  const inputStyle = {
    backgroundColor: 'hsl(var(--card))',
    color: 'hsl(var(--card-foreground))',
    borderColor: 'hsl(var(--border))',
    transition: 'color 250ms ease-in-out, background-color 250ms ease-in-out, border-color 250ms ease-in-out',
  };

  return (
    <div
      className="flex items-center gap-3 rounded-lg px-3 py-2 shadow-sm border"
      style={inputStyle}
    >
      <DatePicker
        selected={start ? new Date(start) : null}
        onChange={(d: Date | null) => {
          if (!d) return;
          const s = d.toISOString().slice(0, 10);
          onChange?.(s, end || s);
        }}
        selectsStart
        startDate={start ? new Date(start) : null}
        endDate={end ? new Date(end) : null}
        className="rounded px-2 py-1 border"
        wrapperClassName="date-picker-wrapper"
        calendarClassName="rounded-lg shadow-lg border"
      />
      <span className="font-bold" style={{ color: 'hsl(var(--muted-foreground))' }}>to</span>
      <DatePicker
        selected={end ? new Date(end) : null}
        onChange={(d: Date | null) => {
          if (!d) return;
          const e = d.toISOString().slice(0, 10);
          onChange?.(start || e, e);
        }}
        selectsEnd
        startDate={start ? new Date(start) : null}
        endDate={end ? new Date(end) : null}
        className="rounded px-2 py-1 border"
        wrapperClassName="date-picker-wrapper"
        calendarClassName="rounded-lg shadow-lg border"
      />
    </div>
  );
}
