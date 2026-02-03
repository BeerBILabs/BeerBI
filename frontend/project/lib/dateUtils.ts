/**
 * Shared date utilities for consistent date handling across the application
 */

/**
 * Format a Date object as YYYY-MM-DD in local timezone
 * Avoids UTC conversion issues that occur with toISOString()
 */
export function formatLocalDate(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Parse a YYYY-MM-DD string into a Date object in local timezone
 * Avoids UTC parsing issues that occur with new Date(string)
 */
export function parseLocalDate(dateStr: string): Date {
  const [year, month, day] = dateStr.split('-').map(Number);
  return new Date(year, month - 1, day);
}

/**
 * Generate an array of date strings (YYYY-MM-DD) between start and end (inclusive)
 */
export function generateDateRange(start: string, end: string): string[] {
  const dates: string[] = [];
  const current = parseLocalDate(start);
  const endDate = parseLocalDate(end);
  
  while (current <= endDate) {
    dates.push(formatLocalDate(current));
    current.setDate(current.getDate() + 1);
  }
  return dates;
}
