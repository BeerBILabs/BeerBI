package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func mustTime(t testing.TB, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}

type beerRow struct {
	giver, recipient, at string
	count                int
}

// statsFixture covers a leap day, a year end and day boundaries.
var statsFixture = []beerRow{
	{"UA", "UB", "2023-12-31T23:59:59Z", 1},
	{"UA", "UC", "2024-01-01T00:00:00Z", 2},
	{"UB", "UA", "2024-02-28T23:59:59Z", 1},
	{"UA", "UB", "2024-02-29T00:00:00Z", 3},
	{"UC", "UB", "2024-02-29T23:59:59Z", 1},
	{"UB", "UC", "2024-03-01T00:00:00Z", 2},
	{"UA", "UB", "2024-04-15T12:00:00Z", 1},
	{"UD", "UA", "2024-12-31T23:59:59Z", 5},
}

func seedStats(t testing.TB, s *SQLiteStore) {
	t.Helper()
	for i, r := range statsFixture {
		at := mustTime(t, r.at)
		if err := s.AddBeer(r.giver, r.recipient, fmt.Sprintf("%d.%06d", at.Unix(), i), at, r.count); err != nil {
			t.Fatalf("add beer: %v", err)
		}
	}
}

func TestDateBounds(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("no tzdata: %v", err)
	}
	cases := []struct {
		name       string
		start, end time.Time
		lo, hi     string
	}{
		{"single day", day("2024-05-10"), day("2024-05-10"), "2024-05-10", "2024-05-11"},
		{"leap day", day("2024-02-29"), day("2024-02-29"), "2024-02-29", "2024-03-01"},
		{"feb non-leap", day("2023-02-01"), day("2023-02-28"), "2023-02-01", "2023-03-01"},
		{"year end", day("2024-12-01"), day("2024-12-31"), "2024-12-01", "2025-01-01"},
		{"zero start", time.Time{}, day("2024-01-01"), "0001-01-01", "2024-01-02"},
		{"time of day ignored", mustTime(t, "2024-05-10T18:30:00Z"), mustTime(t, "2024-05-11T23:59:59Z"), "2024-05-10", "2024-05-12"},
		// The calendar day is taken in each value's own location (as before).
		{"local day", time.Date(2024, 3, 31, 0, 30, 0, 0, berlin), time.Date(2024, 3, 31, 23, 30, 0, 0, berlin), "2024-03-31", "2024-04-01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lo, hi := dateBounds(c.start, c.end)
			if lo != c.lo || hi != c.hi {
				t.Errorf("dateBounds = (%s, %s), want (%s, %s)", lo, hi, c.lo, c.hi)
			}
		})
	}
}

func TestYearBounds(t *testing.T) {
	cases := []struct {
		start, end int
		lo, hi     string
	}{
		{2024, 2024, "2024-01-01", "2025-01-01"},
		{2020, 2025, "2020-01-01", "2026-01-01"},
		{-5, 9999, "0000-01-01", "9999-01-01"},
		{2025, 2024, "2025-01-01", "2025-01-01"}, // empty range
	}
	for _, c := range cases {
		lo, hi := yearBounds(c.start, c.end)
		if lo != c.lo || hi != c.hi {
			t.Errorf("yearBounds(%d, %d) = (%s, %s), want (%s, %s)", c.start, c.end, lo, hi, c.lo, c.hi)
		}
	}
}

func TestCountInDateRange_Boundaries(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	cases := []struct {
		name       string
		count      func(string, time.Time, time.Time) (int, error)
		user       string
		start, end string
		want       int
	}{
		{"given leap day", s.CountGivenInDateRange, "UA", "2024-02-29", "2024-02-29", 3},
		{"given day before leap day", s.CountGivenInDateRange, "UB", "2024-02-28", "2024-02-28", 1},
		{"received leap day both ends", s.CountReceivedInDateRange, "UB", "2024-02-29", "2024-02-29", 4},
		{"received end inclusive", s.CountReceivedInDateRange, "UB", "2024-02-28", "2024-02-29", 4},
		{"given year end", s.CountGivenInDateRange, "UA", "2023-12-31", "2023-12-31", 1},
		{"given new year", s.CountGivenInDateRange, "UA", "2024-01-01", "2024-01-01", 2},
		{"given across year", s.CountGivenInDateRange, "UA", "2023-12-31", "2024-12-31", 7},
		{"received last second of year", s.CountReceivedInDateRange, "UA", "2024-12-31", "2024-12-31", 5},
		{"empty when start after end", s.CountGivenInDateRange, "UA", "2024-03-01", "2024-02-29", 0},
		{"unknown user", s.CountGivenInDateRange, "UZ", "2020-01-01", "2030-01-01", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.count(c.user, day(c.start), day(c.end))
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestCountGivenOnDateAndCountReceived(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	if n, err := s.CountGivenOnDate("UA", "2024-02-29"); err != nil || n != 3 {
		t.Errorf("CountGivenOnDate = %d, %v; want 3", n, err)
	}
	if _, err := s.CountGivenOnDate("UA", "29.02.2024"); err == nil {
		t.Error("CountGivenOnDate accepted an invalid date")
	}
	if n, err := s.CountReceived("UB", ""); err != nil || n != 6 {
		t.Errorf("CountReceived all-time = %d, %v; want 6", n, err)
	}
	if n, err := s.CountReceived("UB", "2024-02-29"); err != nil || n != 4 {
		t.Errorf("CountReceived on day = %d, %v; want 4", n, err)
	}
	if _, err := s.CountReceived("UB", "bad"); err == nil {
		t.Error("CountReceived accepted an invalid date")
	}
}

func TestGetTimelineStats(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	cases := []struct {
		granularity string
		start, end  string
		want        []TimelinePoint
	}{
		{"day", "2024-02-28", "2024-03-01", []TimelinePoint{
			{Date: "2024-02-28", Count: 1},
			{Date: "2024-02-29", Count: 4},
			{Date: "2024-03-01", Count: 2},
		}},
		{"day", "2024-02-29", "2024-02-29", []TimelinePoint{{Date: "2024-02-29", Count: 4}}},
		{"week", "2023-12-31", "2024-01-07", []TimelinePoint{
			{Date: "2023-W52", Count: 1},
			{Date: "2024-W01", Count: 2},
		}},
		{"month", "2023-12-01", "2024-12-31", []TimelinePoint{
			{Date: "2023-12", Count: 1},
			{Date: "2024-01", Count: 2},
			{Date: "2024-02", Count: 5},
			{Date: "2024-03", Count: 2},
			{Date: "2024-04", Count: 1},
			{Date: "2024-12", Count: 5},
		}},
		{"day", "2025-01-01", "2025-12-31", nil},
	}
	for _, c := range cases {
		t.Run(c.granularity+" "+c.start+".."+c.end, func(t *testing.T) {
			got, err := s.GetTimelineStats(day(c.start), day(c.end), c.granularity)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestGetQuarterlyStats(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	got, err := s.GetQuarterlyStats(2023, 2024)
	if err != nil {
		t.Fatal(err)
	}
	want := []QuarterlyStats{
		{Year: 2023, Quarter: 4, Count: 1},
		{Year: 2024, Quarter: 1, Count: 9},
		{Year: 2024, Quarter: 2, Count: 1},
		{Year: 2024, Quarter: 4, Count: 5},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if got, _ := s.GetQuarterlyStats(2024, 2024); len(got) != 3 {
		t.Errorf("2024 only: got %+v, want 3 quarters", got)
	}
	if got, _ := s.GetQuarterlyStats(2025, 2024); len(got) != 0 {
		t.Errorf("reversed years: got %+v, want none", got)
	}
}

func TestGetTopUsers(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	got, err := s.GetTopUsers(day("2023-12-31"), day("2024-12-31"), 10)
	if err != nil {
		t.Fatal(err)
	}
	wantGivers := []TopUserStats{{"UA", 7}, {"UD", 5}, {"UB", 3}, {"UC", 1}}
	// UA and UB tie at 6 received: ties are ordered by user ID.
	wantRecipients := []TopUserStats{{"UA", 6}, {"UB", 6}, {"UC", 4}}
	if !reflect.DeepEqual(got.Givers, wantGivers) {
		t.Errorf("givers = %+v, want %+v", got.Givers, wantGivers)
	}
	if !reflect.DeepEqual(got.Recipients, wantRecipients) {
		t.Errorf("recipients = %+v, want %+v", got.Recipients, wantRecipients)
	}

	limited, err := s.GetTopUsers(day("2023-12-31"), day("2024-12-31"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.Givers) != 2 || len(limited.Recipients) != 2 || limited.Givers[0].UserID != "UA" {
		t.Errorf("limit 2: %+v", limited)
	}

	leap, _ := s.GetTopUsers(day("2024-02-29"), day("2024-02-29"), 10)
	if want := []TopUserStats{{"UA", 3}, {"UC", 1}}; !reflect.DeepEqual(leap.Givers, want) {
		t.Errorf("leap day givers = %+v, want %+v", leap.Givers, want)
	}

	empty, _ := s.GetTopUsers(day("2030-01-01"), day("2030-12-31"), 10)
	if len(empty.Givers) != 0 || len(empty.Recipients) != 0 {
		t.Errorf("empty range: %+v", empty)
	}
}

func TestGetHeatmapStats(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	got, err := s.GetHeatmapStats(day("2023-12-31"), day("2024-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	want := []HeatmapPoint{{"2023-12-31", 1}, {"2024-01-01", 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetPairStats(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	got, err := s.GetPairStats(day("2023-01-01"), day("2024-12-31"), 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []PairStats{
		{Giver: "UA", Recipient: "UB", Count: 5},
		{Giver: "UD", Recipient: "UA", Count: 5},
		{Giver: "UA", Recipient: "UC", Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestGetAllGiversAndRecipientsSorted(t *testing.T) {
	s := newTestStore(t)
	seedStats(t, s)
	givers, err := s.GetAllGivers()
	if err != nil || len(givers) != 4 {
		t.Errorf("givers = %v, %v", givers, err)
	}
	recipients, err := s.GetAllRecipients()
	if err != nil || len(recipients) != 3 {
		t.Errorf("recipients = %v, %v", recipients, err)
	}
}

func TestEventDeduplication(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()

	if ok, _ := s.IsEventProcessed("E1"); ok {
		t.Fatal("E1 processed before marking")
	}
	ok, err := s.TryMarkEventProcessed("E1", now)
	if err != nil || !ok {
		t.Fatalf("first TryMark = %v, %v; want true", ok, err)
	}
	ok, err = s.TryMarkEventProcessed("E1", now)
	if err != nil || ok {
		t.Fatalf("second TryMark = %v, %v; want false", ok, err)
	}
	if ok, err := s.IsEventProcessed("E1"); err != nil || !ok {
		t.Errorf("IsEventProcessed(E1) = %v, %v", ok, err)
	}
	if err := s.MarkEventProcessed("E2", now); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkEventProcessed("E2", now); err != nil {
		t.Errorf("MarkEventProcessed twice: %v", err)
	}
	if ok, _ := s.TryMarkEventProcessed("E2", now); ok {
		t.Error("TryMark after Mark returned true")
	}
}

func TestPruneProcessedEvents(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	for i, age := range []time.Duration{0, 29 * 24 * time.Hour, 31 * 24 * time.Hour, 400 * 24 * time.Hour} {
		if err := s.MarkEventProcessed(fmt.Sprintf("E%d", i), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.PruneProcessedEvents(now.Add(-30 * 24 * time.Hour))
	if err != nil || n != 2 {
		t.Fatalf("pruned %d, %v; want 2", n, err)
	}
	for i, want := range []bool{true, true, false, false} {
		if ok, _ := s.IsEventProcessed(fmt.Sprintf("E%d", i)); ok != want {
			t.Errorf("E%d processed = %v, want %v", i, ok, want)
		}
	}
	// A pruned event can be processed again (Slack never redelivers that late).
	if ok, _ := s.TryMarkEventProcessed("E3", now); !ok {
		t.Error("pruned event could not be re-marked")
	}
}

func TestRunProcessedEventsPruner(t *testing.T) {
	s := newTestStore(t)
	if err := s.MarkEventProcessed("old", time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.RunProcessedEventsPruner(ctx, 10*time.Millisecond, 24*time.Hour)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		ok, err := s.IsEventProcessed("old")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("old event not pruned")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Recent events survive later ticks.
	if err := s.MarkEventProcessed("new", time.Now()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	if ok, _ := s.IsEventProcessed("new"); !ok {
		t.Error("recent event was pruned")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pruner did not stop after cancel")
	}
}

func TestEmojiCounts(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 3; i++ {
		if err := s.IncEmoji("U1", "beer"); err != nil {
			t.Fatal(err)
		}
	}
	if c, err := s.GetCount("U1", "beer"); err != nil || c != 3 {
		t.Errorf("GetCount = %d, %v; want 3", c, err)
	}
	if c, err := s.GetCount("U1", "wine"); err != nil || c != 0 {
		t.Errorf("GetCount missing = %d, %v; want 0", c, err)
	}
}

func TestCachedUser(t *testing.T) {
	s := newTestStore(t)
	if u, err := s.GetCachedUser("U1"); err != nil || u != nil {
		t.Fatalf("missing user = %+v, %v", u, err)
	}
	before := time.Now().Add(-time.Second)
	if err := s.SetCachedUser("U1", "Test User", "https://example.invalid/a.png"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCachedUser("U1", "Renamed User", ""); err != nil {
		t.Fatal(err)
	}
	u, err := s.GetCachedUser("U1")
	if err != nil || u == nil {
		t.Fatalf("get = %+v, %v", u, err)
	}
	if u.RealName != "Renamed User" || u.ProfileImage != "" {
		t.Errorf("upsert not applied: %+v", u)
	}
	if u.UpdatedAt.Before(before) || u.UpdatedAt.After(time.Now().Add(time.Second)) {
		t.Errorf("UpdatedAt = %v, want about now", u.UpdatedAt)
	}
}

func TestAddBeerUpsertsCount(t *testing.T) {
	s := newTestStore(t)
	at := mustTime(t, "2024-06-01T10:00:00Z")
	if err := s.AddBeer("U1", "U2", "1717236000.000001", at, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.AddBeer("U1", "U2", "1717236000.000001", at, 3); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountGivenOnDate("U1", "2024-06-01"); n != 3 {
		t.Errorf("count after upsert = %d, want 3 (last write wins)", n)
	}
	var tsRFC string
	if err := s.db.QueryRow(`SELECT CAST(ts_rfc AS TEXT) FROM beers`).Scan(&tsRFC); err != nil {
		t.Fatal(err)
	}
	if tsRFC != "2024-06-01T10:00:00Z" {
		t.Errorf("ts_rfc = %q", tsRFC)
	}
}
