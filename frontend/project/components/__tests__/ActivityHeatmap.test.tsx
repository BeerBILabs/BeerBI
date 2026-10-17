import { Profiler } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';

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

describe('ActivityHeatmap data loading', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function deferredFetch() {
    const pending: Array<{ url: string; resolve: (body: unknown) => void }> = [];
    const fetchMock = vi.fn((url: string) => {
      return new Promise((resolve) => {
        pending.push({ url, resolve: (body) => resolve({ ok: true, json: async () => body }) });
      });
    });
    vi.stubGlobal('fetch', fetchMock);
    return { fetchMock, pending };
  }

  it('fetches the heatmap through the proxy when no data is preloaded', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => data });
    vi.stubGlobal('fetch', fetchMock);
    render(<ActivityHeatmap startDate="2025-01-01" endDate="2025-01-31" />);

    expect(screen.queryByTestId('heatmap-cell')).toBeNull(); // skeleton while loading
    expect((await screen.findAllByTestId('heatmap-cell')).length).toBe(31);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe('/api/proxy/stats/heatmap?start=2025-01-01&end=2025-01-31');
    const cell = screen.getAllByTestId('heatmap-cell').find((c) => c.getAttribute('data-date') === '2025-01-20')!;
    expect(cell.getAttribute('data-count')).toBe('10');
  });

  it('does not fetch when data is preloaded', () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    renderHeatmap();
    expect(screen.getAllByTestId('heatmap-cell')).toHaveLength(31);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('treats null preloaded data (empty range from the backend) as present: empty grid, no fetch', () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    render(<ActivityHeatmap startDate="2025-01-01" endDate="2025-01-31" preloadedData={null} />);
    const cells = screen.getAllByTestId('heatmap-cell');
    expect(cells).toHaveLength(31);
    expect(cells.every((c) => c.getAttribute('data-count') === '0')).toBe(true);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('shows an error when the request fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }));
    render(<ActivityHeatmap startDate="2025-01-01" endDate="2025-01-31" />);
    expect(await screen.findByText('Failed to fetch heatmap data')).toBeTruthy();
  });

  it('ignores the response of a superseded request', async () => {
    const { fetchMock, pending } = deferredFetch();
    const { rerender } = render(<ActivityHeatmap startDate="2025-01-01" endDate="2025-01-31" />);
    rerender(<ActivityHeatmap startDate="2025-02-01" endDate="2025-02-28" />);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    const signals = fetchMock.mock.calls.map((c) => (c as unknown as [string, RequestInit])[1].signal);
    expect(signals[0]?.aborted).toBe(true);
    expect(signals[1]?.aborted).toBe(false);

    // the newer request answers first, the stale one answers last
    await act(async () => {
      pending[1].resolve([{ date: '2025-02-10', count: 9 }]);
    });
    await act(async () => {
      pending[0].resolve([{ date: '2025-02-10', count: 1 }]);
    });

    const cells = await screen.findAllByTestId('heatmap-cell');
    expect(cells).toHaveLength(28);
    expect(cells.find((c) => c.getAttribute('data-date') === '2025-02-10')?.getAttribute('data-count')).toBe('9');
  });
});
