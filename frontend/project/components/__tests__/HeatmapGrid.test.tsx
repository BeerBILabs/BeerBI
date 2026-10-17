import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { HeatmapGrid } from '../charts/HeatmapGrid';
import { buildHeatmapModel } from '../charts/heatmapModel';

afterEach(() => {
  cleanup();
});

const data = [
  { date: '2025-01-02', count: 3 },
  { date: '2025-01-20', count: 10 },
];

describe('HeatmapGrid memoization', () => {
  // Reading the palette happens once per rendered cell, so counting reads counts cell renders.
  function countingPalette() {
    const counter = { reads: 0 };
    const colors = ['a', 'b', 'c', 'd', 'e'];
    const palette = new Proxy(colors, {
      get(target, prop, receiver) {
        if (typeof prop === 'string' && /^\d$/.test(prop)) counter.reads++;
        return Reflect.get(target, prop, receiver);
      },
    });
    return { palette, counter };
  }

  const model = buildHeatmapModel(data, '2025-01-01', '2025-01-31');
  const onEnter = vi.fn();
  const onLeave = vi.fn();

  function Parent({ palette }: { palette: string[] }) {
    const [n, setN] = useState(0);
    return (
      <div>
        <button onClick={() => setN(n + 1)}>rerender {n}</button>
        <HeatmapGrid
          weeks={model.weeks}
          cells={model.cells}
          palette={palette}
          onCellEnter={onEnter}
          onCellLeave={onLeave}
        />
      </div>
    );
  }

  it('skips rendering the cells when the parent re-renders with unchanged props', () => {
    const { palette, counter } = countingPalette();
    render(<Parent palette={palette} />);
    expect(counter.reads).toBe(31);

    act(() => {
      fireEvent.click(screen.getByRole('button'));
    });
    expect(screen.getByRole('button').textContent).toBe('rerender 1');
    expect(counter.reads).toBe(31);
  });

  it('does re-render the cells when its props change', () => {
    const first = countingPalette();
    const { rerender } = render(<Parent palette={first.palette} />);
    expect(first.counter.reads).toBe(31);

    const second = countingPalette();
    rerender(<Parent palette={second.palette} />);
    expect(second.counter.reads).toBe(31);
  });
});
