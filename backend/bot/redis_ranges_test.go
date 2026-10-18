package main

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestRangesForEvent(t *testing.T) {
	utc := time.UTC
	d := func(y int, m time.Month, day, hour int) time.Time { return time.Date(y, m, day, hour, 0, 0, 0, utc) }

	cases := []struct {
		name  string
		now   time.Time
		event time.Time
		want  []string
	}{
		{
			// C3: in the first days of a month a current beer must not land in last-month/last-quarter.
			name: "beer right now, first day of a quarter", now: d(2026, 10, 1, 9), event: d(2026, 10, 1, 9),
			want: []string{RangeAllTime, RangeToday, RangeLast7Days, RangeCurrentMonth, RangeCurrentQuarter},
		},
		{
			name: "beer right now, first week of a month", now: d(2026, 11, 3, 9), event: d(2026, 11, 3, 8),
			want: []string{RangeAllTime, RangeToday, RangeLast7Days, RangeCurrentMonth, RangeCurrentQuarter},
		},
		{
			name: "beer right now, mid month", now: d(2026, 11, 20, 9), event: d(2026, 11, 20, 9),
			want: []string{RangeAllTime, RangeToday, RangeLast7Days, RangeCurrentMonth, RangeCurrentQuarter},
		},
		{
			name: "late event from yesterday in the same month", now: d(2026, 11, 20, 9), event: d(2026, 11, 19, 23),
			want: []string{RangeAllTime, RangeLast7Days, RangeCurrentMonth, RangeCurrentQuarter},
		},
		{
			name: "event 6 days ago is still in last 7 days", now: d(2026, 11, 20, 9), event: d(2026, 11, 14, 9),
			want: []string{RangeAllTime, RangeLast7Days, RangeCurrentMonth, RangeCurrentQuarter},
		},
		{
			name: "event 7 days ago is not", now: d(2026, 11, 20, 9), event: d(2026, 11, 13, 9),
			want: []string{RangeAllTime, RangeCurrentMonth, RangeCurrentQuarter},
		},
		{
			name: "late event from the previous month, same quarter", now: d(2026, 11, 2, 9), event: d(2026, 10, 31, 22),
			want: []string{RangeAllTime, RangeLast7Days, RangeLastMonth, RangeCurrentQuarter},
		},
		{
			name: "late event from the previous quarter", now: d(2026, 10, 2, 9), event: d(2026, 9, 30, 22),
			want: []string{RangeAllTime, RangeLast7Days, RangeLastMonth, RangeLastQuarter},
		},
		{
			name: "year boundary", now: d(2026, 1, 2, 9), event: d(2025, 12, 31, 22),
			want: []string{RangeAllTime, RangeLast7Days, RangeLastMonth, RangeLastQuarter},
		},
		{
			name: "older event only counts for all-time", now: d(2026, 11, 20, 9), event: d(2026, 2, 3, 9),
			want: []string{RangeAllTime},
		},
		{
			name: "two months back in the same quarter is not last month", now: d(2026, 12, 20, 9), event: d(2026, 10, 5, 9),
			want: []string{RangeAllTime, RangeCurrentQuarter},
		},
		{
			name: "future event (clock skew) counts for its month and quarter", now: d(2026, 11, 20, 9), event: d(2026, 11, 21, 9),
			want: []string{RangeAllTime, RangeCurrentMonth, RangeCurrentQuarter},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rangesForEvent(tc.event, tc.now)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("rangesForEvent = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRangesForEventUsesLocalDays(t *testing.T) {
	berlin := time.FixedZone("CET", 3600)
	now := time.Date(2026, 11, 1, 0, 30, 0, 0, berlin) // 2026-10-31 23:30 UTC
	event := time.Date(2026, 10, 31, 23, 40, 0, 0, time.UTC)

	got := rangesForEvent(event, now) // 00:40 local, 2026-11-01
	if !slices.Contains(got, RangeToday) || !slices.Contains(got, RangeCurrentMonth) || slices.Contains(got, RangeLastMonth) {
		t.Errorf("local calendar day not used: %v", got)
	}
}

func TestCalculateActiveRangesIsForNow(t *testing.T) {
	r := &RedisUserCache{}
	got := r.calculateActiveRanges()
	for _, key := range []string{RangeAllTime, RangeToday, RangeLast7Days, RangeCurrentMonth, RangeCurrentQuarter} {
		if !slices.Contains(got, key) {
			t.Errorf("%s missing in %v", key, got)
		}
	}
	if slices.Contains(got, RangeLastMonth) || slices.Contains(got, RangeLastQuarter) {
		t.Errorf("a beer given now must never count for a past period: %v", got)
	}
}

func TestUntilNextMidnight(t *testing.T) {
	zone := time.FixedZone("CET", 3600)
	cases := []struct {
		now  time.Time
		want time.Duration
	}{
		{time.Date(2026, 10, 18, 23, 59, 30, 0, zone), 30 * time.Second},
		{time.Date(2026, 10, 18, 0, 0, 0, 0, zone), 24 * time.Hour},
		{time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC), 12 * time.Hour},
		{time.Date(2028, 2, 28, 18, 0, 0, 0, time.UTC), 6 * time.Hour}, // into the leap day
	}
	for _, tc := range cases {
		if got := untilNextMidnight(tc.now); got != tc.want {
			t.Errorf("untilNextMidnight(%v) = %v, want %v", tc.now, got, tc.want)
		}
	}
}
