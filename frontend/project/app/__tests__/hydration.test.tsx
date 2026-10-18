import { act, type ReactElement } from 'react';
import { renderToString } from 'react-dom/server';
import { hydrateRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import Home from '../page';
import AnalyticsPage from '../analytics/page';

// `/` and `/analytics` are prerendered at build time (audit F1). The HTML must not depend on the
// date: when the page is served in a later quarter or year than it was built in, hydration has to
// succeed without a mismatch (which would re-render the whole page on the client).

const BUILD_TIME = new Date('2026-08-15T10:00:00'); // Q3 2026
const VISIT_TIME = new Date('2027-01-10T10:00:00'); // Q1 2027: other quarter, other year

let root: Root | null = null;
let container: HTMLElement | null = null;

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] });
  localStorage.clear();
  // data requests never answer: only the shell matters here
  vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})));
});

afterEach(() => {
  act(() => root?.unmount());
  root = null;
  container?.remove();
  container = null;
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

/** Server-render at BUILD_TIME, hydrate at VISIT_TIME; returns the hydration problems. */
async function buildThenHydrate(element: () => ReactElement) {
  vi.setSystemTime(BUILD_TIME);
  const html = renderToString(element());

  container = document.createElement('div');
  container.innerHTML = html;
  document.body.appendChild(container);

  vi.setSystemTime(VISIT_TIME);
  const recoverable = vi.fn();
  const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
  await act(async () => {
    root = hydrateRoot(container!, element(), { onRecoverableError: recoverable });
  });
  const mismatchLogs = consoleError.mock.calls.filter((args) => /hydrat/i.test(args.map(String).join(' ')));
  return { recoverable, mismatchLogs };
}

describe('hydration of prerendered pages after a date change', () => {
  it('detects a date-dependent render (harness check)', async () => {
    function YearNow() {
      return <span>{new Date().getFullYear()}</span>;
    }
    const { recoverable, mismatchLogs } = await buildThenHydrate(() => <YearNow />);
    expect(recoverable.mock.calls.length + mismatchLogs.length).toBeGreaterThan(0);
  });

  it('/ hydrates without mismatch and then shows the current quarter and year', async () => {
    const { recoverable, mismatchLogs } = await buildThenHydrate(() => <Home />);

    expect(recoverable).not.toHaveBeenCalled();
    expect(mismatchLogs).toEqual([]);
    expect(screen.getByRole('tab', { name: 'Q1' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.queryByRole('tab', { name: 'Q3' })).toBeNull();
    expect(screen.getByRole('option', { name: '2027' })).toBeInTheDocument();
    expect(screen.getAllByLabelText('Start date')[0]).toHaveValue('2027-01-01');
  });

  it('/analytics hydrates without mismatch and then uses the current year', async () => {
    const { recoverable, mismatchLogs } = await buildThenHydrate(() => <AnalyticsPage />);

    expect(recoverable).not.toHaveBeenCalled();
    expect(mismatchLogs).toEqual([]);
    expect(screen.getByLabelText('Start date')).toHaveValue('2027-01-01');
    expect(screen.getByLabelText('End date')).toHaveValue('2027-01-10');
  });
});
