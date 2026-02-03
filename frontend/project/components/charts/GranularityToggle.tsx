"use client";

type Granularity = "day" | "week" | "month";

interface GranularityToggleProps {
  value: Granularity;
  onChange: (value: Granularity) => void;
}

const options: { value: Granularity; label: string }[] = [
  { value: "day", label: "Day" },
  { value: "week", label: "Week" },
  { value: "month", label: "Month" },
];

export function GranularityToggle({ value, onChange }: GranularityToggleProps) {
  return (
    <div
      className="inline-flex rounded-lg p-1"
      style={{ backgroundColor: "hsl(var(--muted))" }}
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className="px-3 py-1 text-sm font-medium rounded-md transition-colors"
          style={{
            backgroundColor:
              value === opt.value ? "hsl(var(--background))" : "transparent",
            color:
              value === opt.value
                ? "hsl(var(--foreground))"
                : "hsl(var(--muted-foreground))",
            boxShadow: value === opt.value ? "0 1px 2px rgba(0,0,0,0.1)" : "none",
          }}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

export type { Granularity };
