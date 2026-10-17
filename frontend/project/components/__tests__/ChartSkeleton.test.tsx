import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { ChartContainer, ChartSkeleton } from '../charts/ChartContainer';

afterEach(() => {
  cleanup();
});

describe('ChartSkeleton', () => {
  it('shows the title and subtitle of the chart it stands in for', () => {
    render(<ChartSkeleton title="🤝 Who Appreciates Whom" subtitle="Top giver → recipient pairs" />);
    expect(screen.getByRole('heading', { name: '🤝 Who Appreciates Whom' })).toBeTruthy();
    expect(screen.getByText('Top giver → recipient pairs')).toBeTruthy();
  });

  it('uses the given height so the page does not jump when the chart arrives', () => {
    const { container } = render(<ChartSkeleton title="Activity Over Time" skeletonHeight="h-72" />);
    expect(container.querySelector('.animate-pulse > .h-72')).not.toBeNull();
    expect(container.querySelector('.h-64')).toBeNull();
  });

  it('keeps the default height of h-64 for the regular loading state', () => {
    const { container } = render(
      <ChartContainer title="Top Givers" loading>
        {null}
      </ChartContainer>
    );
    expect(container.querySelector('.animate-pulse > .h-64')).not.toBeNull();
  });
});
