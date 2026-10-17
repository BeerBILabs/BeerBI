import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { RankChangeIndicator } from '../RankChangeIndicator';

afterEach(() => {
  cleanup();
});

// lucide-react 1.x sets aria-hidden on icons unless an a11y prop is passed.
// The rank icons carry meaning, so their aria-label must keep them visible to screen readers.
describe('RankChangeIndicator', () => {
  it.each([
    [5, 5, 'No change'],
    [4, 5, 'Up 1 place'],
    [2, 5, 'Up 3 places'],
    [6, 5, 'Down 1 place'],
    [9, 5, 'Down 4 places'],
  ])('rank %i (previous %i) renders an accessible icon "%s"', (currentRank, previousRank, label) => {
    render(<RankChangeIndicator currentRank={currentRank} previousRank={previousRank} />);
    const icon = screen.getByLabelText(label);
    expect(icon.tagName.toLowerCase()).toBe('svg');
    expect(icon.getAttribute('aria-hidden')).toBeNull();
  });

  it('shows the NEW badge without a previous rank', () => {
    render(<RankChangeIndicator currentRank={1} previousRank={null} />);
    expect(screen.getByLabelText('New this quarter').textContent).toBe('NEW');
  });
});
