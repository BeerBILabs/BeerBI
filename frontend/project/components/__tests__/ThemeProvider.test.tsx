import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, renderHook, act } from '@testing-library/react';
import { ThemeProvider, useTheme } from '../ThemeProvider';

function clearThemeCookie() {
  document.cookie = 'theme=; path=/; max-age=0';
}

beforeEach(() => {
  clearThemeCookie();
  document.documentElement.classList.remove('dark');
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('ThemeProvider', () => {
  it('sets the light theme cookie on the first visit, client-side only', () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch');
    render(<ThemeProvider />);
    expect(document.cookie).toContain('theme=light');
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    // A server action would be a POST to the page; none must happen.
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('writes the cookie with the format the layout and its theme script read', () => {
    const setter = vi.spyOn(document, 'cookie', 'set');
    render(<ThemeProvider />);
    expect(setter).toHaveBeenCalledExactlyOnceWith('theme=light; path=/; max-age=31536000; SameSite=Lax');
  });

  it('leaves an existing dark theme untouched', () => {
    document.cookie = 'theme=dark; path=/';
    document.documentElement.classList.add('dark');
    const setter = vi.spyOn(document, 'cookie', 'set');
    render(<ThemeProvider />);
    expect(setter).not.toHaveBeenCalled();
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('renders nothing', () => {
    const { container } = render(<ThemeProvider />);
    expect(container.innerHTML).toBe('');
  });
});

describe('useTheme', () => {
  it('toggles the dark class and stores the choice in the cookie', () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch');
    const { result } = renderHook(() => useTheme());
    expect(result.current.getTheme()).toBe('light');

    act(() => result.current.toggleTheme());
    expect(result.current.getTheme()).toBe('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.cookie).toContain('theme=dark');

    act(() => result.current.toggleTheme());
    expect(result.current.getTheme()).toBe('light');
    expect(document.cookie).toContain('theme=light');
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('setTheme applies the theme directly', () => {
    const { result } = renderHook(() => useTheme());
    act(() => result.current.setTheme('dark'));
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    act(() => result.current.setTheme('light'));
    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });
});
