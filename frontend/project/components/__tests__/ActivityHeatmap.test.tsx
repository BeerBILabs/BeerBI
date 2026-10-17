import { Profiler } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';

// Wrap the real grid to count how often it is rendered and whether any of its subtree did work.
const counters = vi.hoisted(() => ({ parentRenders: 0, gridCommits: 0 }));

vi.mock('../charts/HeatmapGrid', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../charts/HeatmapGrid')>();
  const Wrapped = (props: React.ComponentProps<typeof actual.HeatmapGrid>) => {
    counters.parentRenders++;
    return (
      <Profiler id="grid" onRender={() => counters.gridCommits++}>
        <actual.HeatmapGrid {...props} />
      </Profiler>
    );
  };
  return { ...actual, HeatmapGrid: Wrapped };
});

import { ActivityHeatmap } from '../charts/ActivityHeatmap';

afterEach(() => {
  cleanup();
  counters.parentRenders = 0;
  counters.gridCommits = 0;
});

const data = [
  { date: '2025-01-02', count: 3 },
  { date: '2025-01-15', count: 1 },
  { date: '2025-01-20', count: 10 },
];

function renderHeatmap() {
  return render(<ActivityHeatmap startDate="2025-01-01" endDate="2025-01-31" preloadedData={data} />);
}

describe('ActivityHeatmap', () => {
  it('renders one cell per day with the right counts', () => {
    renderHeatmap();
    const cells = screen.getAllByTestId('heatmap-cell');
    expect(cells).toHaveLength(31);
    const byDate = new Map(cells.map((c) => [c.getAttribute('data-date'), c.getAttribute('data-count')]));
    expect(byDate.get('2025-01-02')).toBe('3');
    expect(byDate.get('2025-01-20')).toBe('10');
    expect(byDate.get('2025-01-05')).toBe('0');
  });

  it('pads the first and last week with empty placeholders (5 weeks x 7 days)', () => {
    renderHeatmap();
    expect(screen.getAllByTestId('heatmap-empty')).toHaveLength(35 - 31);
  });

  it('shows and hides the tooltip on hover', () => {
    renderHeatmap();
    const cell = screen.getAllByTestId('heatmap-cell').find((c) => c.getAttribute('data-date') === '2025-01-20')!;
    expect(screen.queryByRole('tooltip')).toBeNull();

    fireEvent.mouseEnter(cell);
    const tooltip = screen.getByRole('tooltip');
    expect(tooltip.textContent).toContain('Mon, Jan 20, 2025');
    expect(tooltip.textContent).toContain('10 beers');

    fireEvent.mouseLeave(cell);
    expect(screen.queryByRole('tooltip')).toBeNull();
  });

  it('uses the singular for one beer', () => {
    renderHeatmap();
    const cell = screen.getAllByTestId('heatmap-cell').find((c) => c.getAttribute('data-date') === '2025-01-15')!;
    fireEvent.mouseEnter(cell);
    expect(screen.getByRole('tooltip').textContent).toContain('1 beer');
    expect(screen.getByRole('tooltip').textContent).not.toContain('1 beers');
  });

  it('does not re-render the grid or its cells when hovering', () => {
    renderHeatmap();
    const cells = screen.getAllByTestId('heatmap-cell');
    const parentRenders = counters.parentRenders;
    const gridCommits = counters.gridCommits;
    expect(parentRenders).toBeGreaterThan(0);

    for (const cell of cells.slice(0, 10)) {
      fireEvent.mouseEnter(cell);
      expect(screen.getByRole('tooltip')).toBeTruthy(); // the hover really happened
      fireEvent.mouseLeave(cell);
    }

    expect(counters.parentRenders).toBe(parentRenders);
    expect(counters.gridCommits).toBe(gridCommits);
  });
});
