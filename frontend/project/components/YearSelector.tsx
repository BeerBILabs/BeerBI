"use client";

interface YearSelectorProps {
  selectedYear: number;
  onChange: (year: number) => void;
  disabled?: boolean;
}

export function YearSelector({
  selectedYear,
  onChange,
  disabled = false,
}: YearSelectorProps) {
  const currentYear = new Date().getFullYear();
  const years: number[] = [];
  for (let y = currentYear; y >= 2020; y--) {
    years.push(y);
  }

  return (
    <select
      value={selectedYear}
      onChange={(e) => onChange(parseInt(e.target.value, 10))}
      disabled={disabled}
      className={`px-3 py-1.5 rounded-md text-sm font-medium border cursor-pointer ${
        disabled ? "opacity-50 cursor-not-allowed" : ""
      }`}
      style={{
        backgroundColor: "hsl(var(--card))",
        color: "hsl(var(--card-foreground))",
        borderColor: "hsl(var(--border))",
      }}
      aria-label="Select year"
    >
      {years.map((year) => (
        <option key={year} value={year}>
          {year}
        </option>
      ))}
    </select>
  );
}
