/**
 * Chart theme utilities for Recharts integration with CSS custom properties
 */

// Helper to get computed CSS variable value
function getCSSVariable(name: string): string {
  if (typeof window === 'undefined') return '';
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

// Convert HSL CSS variable to usable color string
function hslVar(name: string): string {
  const value = getCSSVariable(name);
  if (!value) return '';
  return `hsl(${value})`;
}

export interface ChartColors {
  primary: string;
  secondary: string;
  accent: string;
  muted: string;
  background: string;
  foreground: string;
  border: string;
  // Chart-specific colors
  given: string;
  received: string;
  // Heatmap intensity levels (low to high)
  heatmap: string[];
}

// Get theme-aware colors for charts
export function getChartColors(): ChartColors {
  return {
    primary: hslVar('--primary'),
    secondary: hslVar('--secondary'),
    accent: hslVar('--accent'),
    muted: hslVar('--muted'),
    background: hslVar('--background'),
    foreground: hslVar('--foreground'),
    border: hslVar('--border'),
    // Given = primary (golden beer color), Received = accent
    given: hslVar('--primary'),
    received: hslVar('--accent'),
    // Heatmap: 5 intensity levels from muted to primary
    heatmap: [
      hslVar('--muted'),
      'hsl(45, 60%, 70%)',  // Light amber
      'hsl(40, 70%, 55%)',  // Medium amber
      'hsl(35, 80%, 45%)',  // Dark amber
      hslVar('--primary'),  // Full primary
    ],
  };
}

// Default fallback colors (for SSR or when CSS vars unavailable)
export const defaultChartColors: ChartColors = {
  primary: '#D4A84B',
  secondary: '#F5E6C8',
  accent: '#8B6914',
  muted: '#7A6A50',
  background: '#FFFDF5',
  foreground: '#4A3A28',
  border: '#B8A888',
  given: '#D4A84B',
  received: '#8B6914',
  heatmap: [
    '#E8DCC8',
    '#E5C88A',
    '#D4A84B',
    '#B8922E',
    '#8B6914',
  ],
};

// Chart tooltip styles
export const tooltipStyle = {
  backgroundColor: 'hsl(var(--card))',
  border: '1px solid hsl(var(--border))',
  borderRadius: '8px',
  color: 'hsl(var(--foreground))',
  padding: '8px 12px',
};

// Chart grid styles
export const gridStyle = {
  stroke: 'hsl(var(--border))',
  strokeDasharray: '3 3',
};

// Axis styles
export const axisStyle = {
  tick: { fill: 'hsl(var(--muted-foreground))' },
  axisLine: { stroke: 'hsl(var(--border))' },
};

// Format number with K/M suffix
export function formatNumber(value: number): string {
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}K`;
  }
  return value.toString();
}

// Format date for display
export function formatDate(dateStr: string, granularity: 'day' | 'week' | 'month' = 'day'): string {
  if (granularity === 'month') {
    // Format: "Jan 2026"
    const [year, month] = dateStr.split('-');
    const date = new Date(parseInt(year), parseInt(month) - 1);
    return date.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
  }
  if (granularity === 'week') {
    // Format: "W01 2026"
    return dateStr.replace('-W', ' W');
  }
  // Day format: "Jan 15"
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}
