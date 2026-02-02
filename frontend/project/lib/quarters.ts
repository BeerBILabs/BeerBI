/**
 * Quarter utility functions for date calculations.
 * Pure TypeScript, no external dependencies.
 */

/**
 * Get the start and end dates for a given quarter.
 * @param year - The year (e.g., 2026)
 * @param quarter - The quarter (1-4)
 * @returns ISO date strings for quarter start and end
 */
export function getQuarterDates(
  year: number,
  quarter: number
): { start: string; end: string } {
  // Q1=Jan-Mar (0-2), Q2=Apr-Jun (3-5), Q3=Jul-Sep (6-8), Q4=Oct-Dec (9-11)
  const startMonth = (quarter - 1) * 3;
  const start = new Date(year, startMonth, 1);
  // Last day of quarter: day 0 of next month gives last day of previous month
  const end = new Date(year, startMonth + 3, 0);

  return {
    start: start.toISOString().split('T')[0],
    end: end.toISOString().split('T')[0],
  };
}

/**
 * Get the current quarter based on today's date.
 * @returns The current year and quarter (1-4)
 */
export function getCurrentQuarter(): { year: number; quarter: number } {
  const now = new Date();
  const year = now.getFullYear();
  const month = now.getMonth(); // 0-indexed
  const quarter = Math.ceil((month + 1) / 3);

  return { year, quarter };
}

/**
 * Get the previous quarter, handling year wrap.
 * @param year - The current year
 * @param quarter - The current quarter (1-4)
 * @returns The previous quarter's year and quarter
 */
export function getPreviousQuarter(
  year: number,
  quarter: number
): { year: number; quarter: number } {
  if (quarter === 1) {
    return { year: year - 1, quarter: 4 };
  }
  return { year, quarter: quarter - 1 };
}

/**
 * Validate if a year and quarter combination is valid.
 * @param year - The year to validate (must be 2020-2099)
 * @param quarter - The quarter to validate (must be 1-4)
 * @returns true if valid and not in the future
 */
export function isValidQuarter(year: number, quarter: number): boolean {
  // Check basic validity
  if (quarter < 1 || quarter > 4) {
    return false;
  }
  if (year < 2020 || year > 2099) {
    return false;
  }

  // Check if quarter is in the future
  const current = getCurrentQuarter();
  if (year > current.year) {
    return false;
  }
  if (year === current.year && quarter > current.quarter) {
    return false;
  }

  return true;
}

/**
 * Format a quarter number as a label.
 * @param quarter - The quarter (1-4)
 * @returns Formatted label like "Q1", "Q2", etc.
 */
export function formatQuarterLabel(quarter: number): string {
  return `Q${quarter}`;
}
