import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import DateRangePicker from '../DateRangePicker';

afterEach(() => {
  cleanup();
});

function inputs() {
  return {
    start: screen.getByLabelText('Start date') as HTMLInputElement,
    end: screen.getByLabelText('End date') as HTMLInputElement,
  };
}

describe('DateRangePicker', () => {
  it('renders two labelled native date inputs with the given values', () => {
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" />);
    const { start, end } = inputs();
    expect(start.type).toBe('date');
    expect(end.type).toBe('date');
    expect(start.value).toBe('2025-01-10');
    expect(end.value).toBe('2025-02-20');
    expect(screen.getByRole('group', { name: 'Date range' })).toBeTruthy();
  });

  it('renders empty inputs without values', () => {
    render(<DateRangePicker start={null} end={undefined} />);
    const { start, end } = inputs();
    expect(start.value).toBe('');
    expect(end.value).toBe('');
    expect(start.max).toBe('');
    expect(end.min).toBe('');
  });

  it('limits the pickers so that start <= end', () => {
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" />);
    const { start, end } = inputs();
    expect(start.max).toBe('2025-02-20');
    expect(end.min).toBe('2025-01-10');
    expect(start.min).toBe('');
    expect(end.max).toBe('');
  });

  it('emits YYYY-MM-DD with the unchanged end when the start changes', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    fireEvent.change(inputs().start, { target: { value: '2025-01-15' } });
    expect(onChange).toHaveBeenCalledExactlyOnceWith('2025-01-15', '2025-02-20');
  });

  it('emits YYYY-MM-DD with the unchanged start when the end changes', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    fireEvent.change(inputs().end, { target: { value: '2025-03-01' } });
    expect(onChange).toHaveBeenCalledExactlyOnceWith('2025-01-10', '2025-03-01');
  });

  it('uses the picked day for both bounds when the other one is not set yet', () => {
    const onChange = vi.fn();
    const { rerender } = render(<DateRangePicker start={null} end={null} onChange={onChange} />);
    fireEvent.change(inputs().start, { target: { value: '2025-04-02' } });
    expect(onChange).toHaveBeenLastCalledWith('2025-04-02', '2025-04-02');

    rerender(<DateRangePicker start={null} end={null} onChange={onChange} />);
    fireEvent.change(inputs().end, { target: { value: '2025-05-03' } });
    expect(onChange).toHaveBeenLastCalledWith('2025-05-03', '2025-05-03');
  });

  it('applies a typed start after the end only when the field is left, and moves the end along', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    fireEvent.change(inputs().start, { target: { value: '2025-03-05' } });
    expect(onChange).not.toHaveBeenCalled(); // no dragging of the other bound while typing
    expect(inputs().start.value).toBe('2025-03-05');
    fireEvent.blur(inputs().start);
    expect(onChange).toHaveBeenCalledExactlyOnceWith('2025-03-05', '2025-03-05');
  });

  it('applies a typed end before the start on Enter and moves the start along', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    fireEvent.change(inputs().end, { target: { value: '2024-12-24' } });
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.keyDown(inputs().end, { key: 'Enter' });
    expect(onChange).toHaveBeenCalledExactlyOnceWith('2024-12-24', '2024-12-24');
  });

  it('ignores the intermediate years Chrome reports while a year is typed', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    for (const value of ['0002-02-20', '0020-02-20', '0202-02-20']) {
      fireEvent.change(inputs().end, { target: { value } });
      expect(inputs().end.value).toBe(value); // the user still sees what they typed
    }
    expect(onChange).not.toHaveBeenCalled();

    fireEvent.change(inputs().end, { target: { value: '2026-02-20' } });
    expect(onChange).toHaveBeenCalledExactlyOnceWith('2025-01-10', '2026-02-20');
  });

  it('never moves the start while the year of the end is typed', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    for (const value of ['0002-02-20', '0020-02-20', '0202-02-20', '2026-02-20']) {
      fireEvent.change(inputs().end, { target: { value } });
    }
    for (const call of onChange.mock.calls) {
      expect(call[0]).toBe('2025-01-10');
    }
  });

  it('ignores the intermediate years of the start field as well', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2026-02-20" onChange={onChange} />);
    for (const value of ['0002-01-10', '0020-01-10', '0202-01-10']) {
      fireEvent.change(inputs().start, { target: { value } });
    }
    fireEvent.blur(inputs().start);
    expect(onChange).not.toHaveBeenCalled();
    expect(inputs().start.value).toBe('2025-01-10'); // half-typed year: back to the applied date
  });

  it('emits dates as typed, without timezone shifts at month and year ends', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-01" end="2025-12-31" onChange={onChange} />);
    fireEvent.change(inputs().end, { target: { value: '2025-03-31' } });
    expect(onChange).toHaveBeenLastCalledWith('2025-01-01', '2025-03-31');
  });

  it('ignores a cleared field and restores the applied date when it is left', () => {
    const onChange = vi.fn();
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" onChange={onChange} />);
    fireEvent.change(inputs().start, { target: { value: '' } });
    expect(onChange).not.toHaveBeenCalled();
    expect(inputs().start.value).toBe('');
    fireEvent.blur(inputs().start);
    expect(inputs().start.value).toBe('2025-01-10');
    expect(onChange).not.toHaveBeenCalled();
  });

  it('follows the props when the parent changes the range', () => {
    const { rerender } = render(<DateRangePicker start="2025-01-10" end="2025-02-20" />);
    fireEvent.change(inputs().start, { target: { value: '' } });
    rerender(<DateRangePicker start="2025-06-01" end="2025-06-30" />);
    expect(inputs().start.value).toBe('2025-06-01');
    expect(inputs().end.value).toBe('2025-06-30');
  });

  it('keeps the field names of the previous picker', () => {
    render(<DateRangePicker start="2025-01-10" end="2025-02-20" />);
    expect(inputs().start.name).toBe('date-from');
    expect(inputs().end.name).toBe('date-to');
  });

  it('does not hardcode colors', () => {
    const { container } = render(<DateRangePicker start="2025-01-10" end="2025-02-20" />);
    const styles = Array.from(container.querySelectorAll('[style]')).map((el) => el.getAttribute('style') ?? '');
    expect(styles.length).toBeGreaterThan(0);
    for (const style of styles) {
      expect(style).not.toMatch(/#[0-9a-f]{3,8}\b|rgb\(/i);
    }
  });
});
