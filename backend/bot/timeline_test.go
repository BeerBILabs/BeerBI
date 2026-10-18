package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// The timeline API has a single series: {"date", "count"} (B4).
func TestTimelineHandler_SingleSeries(t *testing.T) {
	s := newTestStore(t)
	at := time.Date(2024, 2, 29, 10, 0, 0, 0, time.UTC)
	for i, c := range []int{2, 3} {
		if err := s.AddBeer("UA", "UB", fmt.Sprintf("1709200800.%06d", i), at, c); err != nil {
			t.Fatal(err)
		}
	}
	h := NewAPIHandlers(s, nil, nil, nil, zerolog.Nop())

	rec := httptest.NewRecorder()
	h.TimelineHandler(rec, httptest.NewRequest(http.MethodGet, "/api/stats/timeline?start=2024-02-28&end=2024-03-01&granularity=day", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	if got, want := strings.TrimSpace(rec.Body.String()), `[{"date":"2024-02-29","count":5}]`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}

	for _, q := range []string{"start=2024-02-28&end=2024-03-01&granularity=year", "start=bad&end=2024-03-01"} {
		rec := httptest.NewRecorder()
		h.TimelineHandler(rec, httptest.NewRequest(http.MethodGet, "/api/stats/timeline?"+q, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", q, rec.Code)
		}
	}
}

// The combined analytics response carries the same single-series points:
// every timeline entry has exactly the keys "date" and "count".
func TestCombinedAnalyticsHandler_TimelineKeys(t *testing.T) {
	s := newTestStore(t)
	at := time.Date(2024, 2, 29, 10, 0, 0, 0, time.UTC)
	if err := s.AddBeer("UA", "UB", "1709200800.000001", at, 4); err != nil {
		t.Fatal(err)
	}
	h := NewAPIHandlers(s, nil, nil, nil, zerolog.Nop())

	rec := httptest.NewRecorder()
	h.CombinedAnalyticsHandler(rec, httptest.NewRequest(http.MethodGet, "/api/stats/combined?start=2024-02-01&end=2024-02-29", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	var body struct {
		Timeline []map[string]any `json:"timeline"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Timeline) != 1 {
		t.Fatalf("timeline = %v, want one point", body.Timeline)
	}
	p := body.Timeline[0]
	keys := slices.Sorted(maps.Keys(p))
	if !slices.Equal(keys, []string{"count", "date"}) {
		t.Errorf("timeline[0] keys = %v, want [count date]", keys)
	}
	if p["date"] != "2024-02-29" || p["count"] != float64(4) {
		t.Errorf("timeline[0] = %v, want date 2024-02-29, count 4", p)
	}
}

func TestParseRetentionDays(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"", DefaultProcessedEventsRetentionDays, false},
		{"  ", DefaultProcessedEventsRetentionDays, false},
		{"7", 7, false},
		{" 90 ", 90, false},
		{"1", 1, false},
		{"0", 0, true},
		{"-3", 0, true},
		{"30d", 0, true},
		{"1.5", 0, true},
		{"3650", 3650, false},
		{"3651", 0, true},
		{"106752", 0, true}, // would overflow time.Duration as days*24h
	}
	for _, c := range cases {
		got, err := ParseRetentionDays(c.in)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("ParseRetentionDays(%q) = %d, %v; want %d, err=%v", c.in, got, err, c.want, c.wantErr)
		}
	}
}
