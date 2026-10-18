package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseDateRangeFromParams(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantStart string
		wantEnd   string
		wantErr   bool
	}{
		{name: "single day", query: "day=2026-03-04", wantStart: "2026-03-04", wantEnd: "2026-03-04"},
		{name: "start and end", query: "start=2026-01-01&end=2026-03-31", wantStart: "2026-01-01", wantEnd: "2026-03-31"},
		{name: "start equals end", query: "start=2026-02-02&end=2026-02-02", wantStart: "2026-02-02", wantEnd: "2026-02-02"},
		{name: "leap day", query: "day=2028-02-29", wantStart: "2028-02-29", wantEnd: "2028-02-29"},
		{name: "day wins over range", query: "day=2026-05-05&start=2026-01-01&end=2026-12-31", wantStart: "2026-05-05", wantEnd: "2026-05-05"},
		{name: "nothing given", query: "", wantErr: true},
		{name: "only start", query: "start=2026-01-01", wantErr: true},
		{name: "only end", query: "end=2026-01-01", wantErr: true},
		{name: "invalid day", query: "day=2026-13-01", wantErr: true},
		{name: "not a date", query: "day=yesterday", wantErr: true},
		{name: "invalid start", query: "start=nope&end=2026-01-01", wantErr: true},
		{name: "invalid end", query: "start=2026-01-01&end=2026-02-30", wantErr: true},
		{name: "start after end", query: "start=2026-03-01&end=2026-02-01", wantErr: true},
		{name: "non-leap day", query: "day=2026-02-29", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/x?"+tc.query, nil)
			start, end, err := parseDateRangeFromParams(r)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %v..%v", start, end)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := start.Format(dateLayout); got != tc.wantStart {
				t.Errorf("start = %s, want %s", got, tc.wantStart)
			}
			if got := end.Format(dateLayout); got != tc.wantEnd {
				t.Errorf("end = %s, want %s", got, tc.wantEnd)
			}
		})
	}
}

func TestParseSlackTimestamp(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{name: "with fraction", in: "1700000000.123456", want: time.Unix(1700000000, 123456000)},
		{name: "short fraction is padded", in: "1700000000.5", want: time.Unix(1700000000, 500000000)},
		{name: "long fraction is trimmed", in: "1700000000.1234567891", want: time.Unix(1700000000, 123456789)},
		{name: "without fraction", in: "1700000000", want: time.Unix(1700000000, 0)},
		{name: "empty fraction", in: "1700000000.", want: time.Unix(1700000000, 0)},
		{name: "garbage fraction is ignored", in: "1700000000.xx", want: time.Unix(1700000000, 0)},
		{name: "garbage seconds", in: "abc.123", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSlackTimestamp(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQuarterHelpers(t *testing.T) {
	wantQuarter := map[time.Month]int{
		time.January: 1, time.February: 1, time.March: 1,
		time.April: 2, time.May: 2, time.June: 2,
		time.July: 3, time.August: 3, time.September: 3,
		time.October: 4, time.November: 4, time.December: 4,
	}
	for month, want := range wantQuarter {
		if got := getQuarterNumber(time.Date(2026, month, 15, 0, 0, 0, 0, time.UTC)); got != want {
			t.Errorf("getQuarterNumber(%v) = %d, want %d", month, got, want)
		}
	}
	wantStart := map[int]time.Month{1: time.January, 2: time.April, 3: time.July, 4: time.October}
	for quarter, want := range wantStart {
		if got := getQuarterStartMonth(quarter); got != want {
			t.Errorf("getQuarterStartMonth(%d) = %v, want %v", quarter, got, want)
		}
	}
}
