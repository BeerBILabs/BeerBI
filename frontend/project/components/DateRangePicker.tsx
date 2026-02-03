"use client"
import type { ReactElement } from 'react'
import DatePicker from 'react-datepicker'
import 'react-datepicker/dist/react-datepicker.css'
import { parseLocalDate, formatLocalDate } from '@/lib/dateUtils'

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
      className="flex flex-wrap items-center gap-2 sm:gap-3 rounded-lg px-3 py-2 shadow-sm border w-full max-w-full"
      style={inputStyle}
    >
      <DatePicker
        selected={start ? parseLocalDate(start) : null}
        onChange={(d: Date | null) => {
          if (!d) return;
          const s = formatLocalDate(d);
          onChange?.(s, end || s);
        }}
        selectsStart
        startDate={start ? parseLocalDate(start) : null}
        endDate={end ? parseLocalDate(end) : null}
        className="rounded px-2 py-1 cursor-pointer w-full min-w-0 text-sm"
        wrapperClassName="date-picker-wrapper flex-1 min-w-[100px]"
        calendarClassName="rounded-lg shadow-lg"
        name="date-from"
      />
      <span className="font-bold" style={{ color: 'hsl(var(--muted-foreground))' }}>to</span>
      <DatePicker
        selected={end ? parseLocalDate(end) : null}
        onChange={(d: Date | null) => {
          if (!d) return;
          const e = formatLocalDate(d);
          onChange?.(start || e, e);
        }}
        selectsEnd
        startDate={start ? parseLocalDate(start) : null}
        endDate={end ? parseLocalDate(end) : null}
        className="rounded px-2 py-1 cursor-pointer w-full min-w-0 text-sm"
        wrapperClassName="date-picker-wrapper flex-1 min-w-[100px]"
        calendarClassName="rounded-lg shadow-lg"
        name="date-to"
      />
    </div>
  );
}
