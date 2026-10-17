import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import ThemeToggle from '../ThemeToggle';

beforeEach(() => {
  document.cookie = 'theme=; path=/; max-age=0';
  document.documentElement.classList.remove('dark');
});

afterEach(() => {
  cleanup();
});

describe('ThemeToggle', () => {
  it('offers dark mode in the light theme and switches both ways', async () => {
    render(<ThemeToggle />);
    const button = await screen.findByRole('button', { name: 'Switch to dark mode' });

    fireEvent.click(button);
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.cookie).toContain('theme=dark');
    expect(await screen.findByRole('button', { name: 'Switch to light mode' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Switch to light mode' }));
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(await screen.findByRole('button', { name: 'Switch to dark mode' })).toBeTruthy();
  });

  it('starts in the dark state when the page is already dark', async () => {
    document.documentElement.classList.add('dark');
    render(<ThemeToggle />);
    expect(await screen.findByRole('button', { name: 'Switch to light mode' })).toBeTruthy();
  });

  it('follows theme changes made elsewhere', async () => {
    render(<ThemeToggle />);
    await screen.findByRole('button', { name: 'Switch to dark mode' });
    document.documentElement.classList.add('dark');
    expect(await screen.findByRole('button', { name: 'Switch to light mode' })).toBeTruthy();
  });

  it('shows the tooltip on hover', async () => {
    render(<ThemeToggle />);
    const button = await screen.findByRole('button', { name: 'Switch to dark mode' });
    fireEvent.mouseEnter(button);
    expect(screen.getByRole('tooltip').textContent).toBe('Switch to dark mode');
    fireEvent.mouseLeave(button);
    expect(screen.queryByRole('tooltip')).toBeNull();
  });
});
