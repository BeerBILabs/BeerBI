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
  return (
    <div className="flex items-center gap-3 card rounded-lg px-3 py-2 shadow-sm border bordered">
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
        className="rounded bordered card px-2 py-1 transition"
        calendarClassName="rounded-lg shadow-lg border bordered card"
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
        className="rounded bordered card px-2 py-1 transition"
        calendarClassName="rounded-lg shadow-lg border bordered card"
      />
    </div>
  );
}
