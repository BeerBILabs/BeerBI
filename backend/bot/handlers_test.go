package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// Test data: a fixed clock and a handful of beers with made-up user IDs.
const (
	hGiver1 = "UGIVERONE"
	hGiver2 = "UGIVERTWO"
	hRecvA  = "URECVAAAA"
	hRecvB  = "URECVBBBB"
)

var handlerTestNow = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

// newStatsTestHandlers returns handlers on a fresh store with seeded beers and a fixed clock.
// Slack and Redis are not configured (the stats handlers need neither).
func newStatsTestHandlers(t *testing.T) (*APIHandlers, *SQLiteStore) {
	t.Helper()
	store := newTestStore(t)

	beers := []struct {
		giver, recipient string
		at               time.Time
		count            int
	}{
		{hGiver1, hRecvA, time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC), 2},
		{hGiver2, hRecvA, time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC), 1},
		{hGiver1, hRecvB, time.Date(2026, 7, 5, 18, 30, 0, 0, time.UTC), 3},
		{hGiver1, hRecvA, time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC), 4},
	}
	for i, b := range beers {
		ts := b.at.Format("20060102150405") + ".00000" + string(rune('0'+i))
		if err := store.AddBeer(b.giver, b.recipient, ts, b.at, b.count); err != nil {
			t.Fatalf("seed beer %d: %v", i, err)
		}
	}

	h := NewAPIHandlers(store, nil, nil, nil, zerolog.Nop())
	h.now = func() time.Time { return handlerTestNow }
	return h, store
}

// serve calls handler with a GET request and returns the recorded response.
func serve(handler http.HandlerFunc, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body is not valid JSON (%v): %q", err, rec.Body.String())
	}
	return v
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, want, rec.Body.String())
	}
}

func TestGivenAndReceivedHandlers(t *testing.T) {
	h, _ := newStatsTestHandlers(t)

	rec := serve(h.GivenHandler, "/api/given?user="+hGiver1+"&start=2026-10-01&end=2026-10-31")
	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	given := decodeBody[givenResponse](t, rec)
	if want := (givenResponse{User: hGiver1, Start: "2026-10-01", End: "2026-10-31", Given: 2}); given != want {
		t.Errorf("given = %+v, want %+v", given, want)
	}

	rec = serve(h.GivenHandler, "/api/given?user="+hGiver1+"&day=2026-07-05")
	wantStatus(t, rec, http.StatusOK)
	if got := decodeBody[givenResponse](t, rec).Given; got != 3 {
		t.Errorf("given on day = %d, want 3", got)
	}

	rec = serve(h.ReceivedHandler, "/api/received?user="+hRecvA+"&start=2026-01-01&end=2026-12-31")
	wantStatus(t, rec, http.StatusOK)
	received := decodeBody[receivedResponse](t, rec)
	if want := (receivedResponse{User: hRecvA, Start: "2026-01-01", End: "2026-12-31", Received: 3}); received != want {
		t.Errorf("received = %+v, want %+v", received, want)
	}

	// A user without beers counts zero, not an error.
	rec = serve(h.ReceivedHandler, "/api/received?user=UNOBODY&day=2026-10-18")
	wantStatus(t, rec, http.StatusOK)
	if got := decodeBody[receivedResponse](t, rec).Received; got != 0 {
		t.Errorf("received = %d, want 0", got)
	}
}

// C4: the user ID is echoed into JSON; it must be escaped, not formatted into a string.
func TestGivenHandlerEscapesUser(t *testing.T) {
	h, _ := newStatsTestHandlers(t)
	user := `U"},"given":999,"x":{"`
	rec := serve(h.GivenHandler, "/api/given?day=2026-10-18&user="+url.QueryEscape(user))
	wantStatus(t, rec, http.StatusOK)
	got := decodeBody[givenResponse](t, rec)
	if got.User != user || got.Given != 0 {
		t.Errorf("got %+v, want user %q and 0 beers", got, user)
	}
}

func TestGivenReceivedBadRequests(t *testing.T) {
	h, _ := newStatsTestHandlers(t)
	cases := []struct {
		name    string
		handler http.HandlerFunc
		target  string
	}{
		{"given: no user", h.GivenHandler, "/api/given?day=2026-10-18"},
		{"given: no range", h.GivenHandler, "/api/given?user=" + hGiver1},
		{"given: bad date", h.GivenHandler, "/api/given?user=" + hGiver1 + "&day=18.10.2026"},
		{"given: start after end", h.GivenHandler, "/api/given?user=" + hGiver1 + "&start=2026-10-18&end=2026-10-01"},
		{"received: no user", h.ReceivedHandler, "/api/received?day=2026-10-18"},
		{"received: no range", h.ReceivedHandler, "/api/received?user=" + hRecvA},
		{"received: bad date", h.ReceivedHandler, "/api/received?user=" + hRecvA + "&start=x&end=y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantStatus(t, serve(tc.handler, tc.target), http.StatusBadRequest)
		})
	}
}

func TestTimelineHandler(t *testing.T) {
	h, _ := newStatsTestHandlers(t)

	cases := []struct {
		query      string
		wantPoints int
	}{
		{"start=2026-10-01&end=2026-10-31", 2},                   // default granularity: day
		{"start=2026-10-01&end=2026-10-31&granularity=day", 2},   // 17th and 18th
		{"start=2026-07-01&end=2026-10-31&granularity=month", 2}, // 2026-07, 2026-10
		{"start=2026-10-01&end=2026-10-31&granularity=week", 1},  // both days in one week
		{"start=2027-01-01&end=2027-01-31&granularity=day", 0},   // empty range
		{"start=2025-12-31&end=2026-10-17&granularity=month", 3}, // year boundary: 2025-12, 2026-07, 2026-10
		{"day=2026-10-18", 1},                                    // single day
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			rec := serve(h.TimelineHandler, "/api/stats/timeline?"+tc.query)
			wantStatus(t, rec, http.StatusOK)
			points := decodeBody[[]map[string]any](t, rec)
			if len(points) != tc.wantPoints {
				t.Fatalf("got %d points, want %d: %v", len(points), tc.wantPoints, points)
			}
			for _, p := range points {
				if p["date"] == "" || p["date"] == nil {
					t.Errorf("point without date: %v", p)
				}
			}
		})
	}

	for _, q := range []string{
		"start=2026-10-01&end=2026-10-31&granularity=year",
		"start=2026-10-01&end=2026-10-31&granularity=DAY",
		"granularity=day",
		"start=bad&end=2026-10-31",
	} {
		wantStatus(t, serve(h.TimelineHandler, "/api/stats/timeline?"+q), http.StatusBadRequest)
	}
}

func TestQuarterlyHandler(t *testing.T) {
	h, _ := newStatsTestHandlers(t)

	rec := serve(h.QuarterlyHandler, "/api/stats/quarterly?start_year=2025&end_year=2026")
	wantStatus(t, rec, http.StatusOK)
	got := map[[2]int]int{}
	for _, q := range decodeBody[[]QuarterlyStats](t, rec) {
		got[[2]int{q.Year, q.Quarter}] = q.Count
	}
	want := map[[2]int]int{{2025, 4}: 4, {2026, 3}: 3, {2026, 4}: 3}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("quarter %v = %d, want %d", k, got[k], v)
		}
	}

	// Defaults (clock year and the two before) and unparsable values fall back to them.
	for _, q := range []string{"", "start_year=abc&end_year=", "start_year=2025"} {
		wantStatus(t, serve(h.QuarterlyHandler, "/api/stats/quarterly?"+q), http.StatusOK)
	}
	startYear, endYear := h.parseYearRange(httptest.NewRequest(http.MethodGet, "/x?start_year=abc", nil))
	if startYear != 2024 || endYear != 2026 {
		t.Errorf("parseYearRange defaults = %d..%d, want 2024..2026", startYear, endYear)
	}
}

func TestTopUsersHandler(t *testing.T) {
	h, _ := newStatsTestHandlers(t)

	rec := serve(h.TopUsersHandler, "/api/stats/top?start=2026-01-01&end=2026-12-31")
	wantStatus(t, rec, http.StatusOK)
	top := decodeBody[TopUsersResult](t, rec)
	if len(top.Givers) != 2 || top.Givers[0] != (TopUserStats{UserID: hGiver1, Count: 5}) || top.Givers[1] != (TopUserStats{UserID: hGiver2, Count: 1}) {
		t.Errorf("givers = %+v", top.Givers)
	}
	if len(top.Recipients) != 2 {
		t.Errorf("recipients = %+v", top.Recipients)
	}

	// limit is applied, and an out-of-range limit falls back to the default instead of failing.
	rec = serve(h.TopUsersHandler, "/api/stats/top?start=2026-01-01&end=2026-12-31&limit=1")
	if got := decodeBody[TopUsersResult](t, rec); len(got.Givers) != 1 || got.Givers[0].UserID != hGiver1 {
		t.Errorf("limit=1 givers = %+v", got.Givers)
	}
	for _, limit := range []string{"0", "-3", "101", "abc"} {
		rec = serve(h.TopUsersHandler, "/api/stats/top?start=2026-01-01&end=2026-12-31&limit="+limit)
		wantStatus(t, rec, http.StatusOK)
		if got := decodeBody[TopUsersResult](t, rec); len(got.Givers) != 2 {
			t.Errorf("limit=%s: givers = %+v, want the default limit (all 2)", limit, got.Givers)
		}
	}

	wantStatus(t, serve(h.TopUsersHandler, "/api/stats/top"), http.StatusBadRequest)
}

func TestHeatmapAndPairsHandlers(t *testing.T) {
	h, _ := newStatsTestHandlers(t)

	rec := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-10-01&end=2026-10-31")
	wantStatus(t, rec, http.StatusOK)
	days := map[string]int{}
	for _, p := range decodeBody[[]HeatmapPoint](t, rec) {
		days[p.Date] = p.Count
	}
	if len(days) != 2 || days["2026-10-17"] != 2 || days["2026-10-18"] != 1 {
		t.Errorf("heatmap = %v", days)
	}
	wantStatus(t, serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-10-31&end=2026-10-01"), http.StatusBadRequest)

	rec = serve(h.PairsHandler, "/api/stats/pairs?start=2026-01-01&end=2026-12-31")
	wantStatus(t, rec, http.StatusOK)
	pairs := decodeBody[[]PairStats](t, rec)
	if len(pairs) != 3 || pairs[0] != (PairStats{Giver: hGiver1, Recipient: hRecvB, Count: 3}) {
		t.Errorf("pairs = %+v", pairs)
	}
	rec = serve(h.PairsHandler, "/api/stats/pairs?start=2026-01-01&end=2026-12-31&limit=1")
	if got := decodeBody[[]PairStats](t, rec); len(got) != 1 {
		t.Errorf("limit=1 pairs = %+v", got)
	}
	rec = serve(h.PairsHandler, "/api/stats/pairs?start=2026-01-01&end=2026-12-31&limit=500")
	if got := decodeBody[[]PairStats](t, rec); len(got) != 3 {
		t.Errorf("limit=500 pairs = %+v, want default limit (all 3)", got)
	}
	wantStatus(t, serve(h.PairsHandler, "/api/stats/pairs"), http.StatusBadRequest)
}

func TestCombinedAnalyticsHandler(t *testing.T) {
	h, _ := newStatsTestHandlers(t)

	rec := serve(h.CombinedAnalyticsHandler, "/api/stats/combined?start=2026-01-01&end=2026-12-31&granularity=month&start_year=2025&end_year=2026")
	wantStatus(t, rec, http.StatusOK)
	got := decodeBody[CombinedAnalyticsResponse](t, rec)

	if len(got.Timeline) != 2 || len(got.Heatmap) != 3 || len(got.Pairs) != 3 || len(got.Quarterly) != 3 {
		t.Errorf("combined = %+v", got)
	}
	if len(got.TopGivers) != 2 || got.TopGivers[0].UserID != hGiver1 || len(got.TopRecipients) != 2 {
		t.Errorf("top users = %+v / %+v", got.TopGivers, got.TopRecipients)
	}

	// Same data as the single endpoints.
	single := decodeBody[[]PairStats](t, serve(h.PairsHandler, "/api/stats/pairs?start=2026-01-01&end=2026-12-31&limit=15"))
	if len(single) != len(got.Pairs) || single[0] != got.Pairs[0] {
		t.Errorf("pairs differ: %+v vs %+v", single, got.Pairs)
	}

	// pairs_limit and limit are honoured and clamped like in the single endpoints.
	rec = serve(h.CombinedAnalyticsHandler, "/api/stats/combined?start=2026-01-01&end=2026-12-31&limit=1&pairs_limit=1")
	got = decodeBody[CombinedAnalyticsResponse](t, rec)
	if len(got.TopGivers) != 1 || len(got.Pairs) != 1 {
		t.Errorf("limits not applied: %d givers, %d pairs", len(got.TopGivers), len(got.Pairs))
	}
	rec = serve(h.CombinedAnalyticsHandler, "/api/stats/combined?start=2026-01-01&end=2026-12-31&limit=1000&pairs_limit=51")
	got = decodeBody[CombinedAnalyticsResponse](t, rec)
	if len(got.TopGivers) != 2 || len(got.Pairs) != 3 {
		t.Errorf("out-of-range limits must fall back to the defaults: %d givers, %d pairs", len(got.TopGivers), len(got.Pairs))
	}

	for _, q := range []string{"", "start=2026-01-01&end=2026-12-31&granularity=hour", "start=x&end=y"} {
		wantStatus(t, serve(h.CombinedAnalyticsHandler, "/api/stats/combined?"+q), http.StatusBadRequest)
	}
}

func TestHandlersReturn500WhenDatabaseFails(t *testing.T) {
	h, store := newStatsTestHandlers(t)
	if err := store.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	rng := "start=2026-01-01&end=2026-12-31"
	cases := map[string]http.HandlerFunc{
		"/api/given?user=U1&" + rng:    h.GivenHandler,
		"/api/received?user=U1&" + rng: h.ReceivedHandler,
		"/api/stats/timeline?" + rng:   h.TimelineHandler,
		"/api/stats/quarterly":         h.QuarterlyHandler,
		"/api/stats/top?" + rng:        h.TopUsersHandler,
		"/api/stats/heatmap?" + rng:    h.HeatmapHandler,
		"/api/stats/pairs?" + rng:      h.PairsHandler,
		"/api/stats/combined?" + rng:   h.CombinedAnalyticsHandler,
	}
	for target, handler := range cases {
		rec := serve(handler, target)
		wantStatus(t, rec, http.StatusInternalServerError)
		if rec.Header().Get("ETag") != "" || strings.Contains(rec.Header().Get("Cache-Control"), "max-age") {
			t.Errorf("%s: an error response must not be cacheable: %v", target, rec.Header())
		}
	}
}

func TestHealthHandler(t *testing.T) {
	var authOK atomic.Bool
	authOK.Store(true)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if authOK.Load() {
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"UBOT","team_id":"T0TEST"}`))
		} else {
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
		}
	}))
	t.Cleanup(api.Close)
	client := slack.New("xoxb-test", slack.OptionAPIURL(api.URL+"/"))
	manager := &SlackConnectionManager{client: client, logger: zerolog.Nop(), isConnected: true}
	h := NewAPIHandlers(nil, client, manager, nil, zerolog.Nop())

	rec := serve(h.HealthHandler, "/api/health")
	wantStatus(t, rec, http.StatusOK)
	health := decodeBody[map[string]any](t, rec)
	if health["status"] != "healthy" || health["slack_connected"] != true || health["service"] != "beerbot-backend" {
		t.Errorf("health = %v", health)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("health must not be cached, Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
	if _, err := time.Parse(time.RFC3339, health["timestamp"].(string)); err != nil {
		t.Errorf("timestamp: %v", err)
	}

	wantStatus(t, serve(h.HealthHandler, "/api/health?check_slack=true"), http.StatusOK)

	authOK.Store(false)
	rec = serve(h.HealthHandler, "/api/health?check_slack=true")
	wantStatus(t, rec, http.StatusServiceUnavailable)
	health = decodeBody[map[string]any](t, rec)
	if health["status"] != "degraded" || health["slack_connection_error"] == nil {
		t.Errorf("degraded health = %v", health)
	}
}

func TestMatchDateRangeToCache(t *testing.T) {
	d := func(s string) time.Time {
		v, err := time.Parse(dateLayout, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := []struct {
		name       string
		now        string
		start, end string
		want       string
	}{
		{"today", "2026-10-18", "2026-10-18", "2026-10-18", RangeToday},
		{"last 7 days", "2026-10-18", "2026-10-12", "2026-10-18", RangeLast7Days},
		{"8 days is not last 7 days", "2026-10-18", "2026-10-11", "2026-10-18", ""},
		{"current month", "2026-10-18", "2026-10-01", "2026-10-18", RangeCurrentMonth},
		{"month so far, not up to today", "2026-10-18", "2026-10-01", "2026-10-17", ""},
		{"last month", "2026-10-18", "2026-09-01", "2026-09-30", RangeLastMonth},
		{"current quarter", "2026-11-20", "2026-10-01", "2026-11-20", RangeCurrentQuarter},
		{"current month in second month of quarter", "2026-11-20", "2026-11-01", "2026-11-20", RangeCurrentMonth},
		{"last quarter", "2026-11-20", "2026-07-01", "2026-09-30", RangeLastQuarter},
		{"last quarter across new year", "2026-02-10", "2025-10-01", "2025-12-31", RangeLastQuarter},
		{"last month across new year", "2026-01-05", "2025-12-01", "2025-12-31", RangeLastMonth},
		{"december is last month and not last quarter in january", "2026-01-05", "2025-10-01", "2025-12-31", RangeLastQuarter},
		{"last quarter ends before the quarter starts", "2026-04-01", "2026-01-01", "2026-03-31", RangeLastQuarter},
		{"leap day last month", "2028-03-10", "2028-02-01", "2028-02-29", RangeLastMonth},
		{"all time", "2026-10-18", "2019-01-01", "2026-10-18", RangeAllTime},
		{"all time from 2020", "2026-10-18", "2020-06-01", "2026-10-18", RangeAllTime},
		{"since 2021 is custom", "2026-10-18", "2021-01-01", "2026-10-18", ""},
		{"all time must end today", "2026-10-18", "2019-01-01", "2026-10-17", ""},
		{"arbitrary range", "2026-10-18", "2026-03-03", "2026-05-05", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &APIHandlers{now: func() time.Time { return d(tc.now).Add(15 * time.Hour) }}
			if got := h.matchDateRangeToCache(d(tc.start), d(tc.end)); got != tc.want {
				t.Errorf("matchDateRangeToCache(%s, %s) at %s = %q, want %q", tc.start, tc.end, tc.now, got, tc.want)
			}
		})
	}
}

func TestCacheHeadersAndETag(t *testing.T) {
	h, _ := newStatsTestHandlers(t) // clock: 2026-10-18

	closed := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-07-01&end=2026-09-30")
	wantStatus(t, closed, http.StatusOK)
	if got := closed.Header().Get("Cache-Control"); got != "private, max-age=3600" {
		t.Errorf("closed period Cache-Control = %q", got)
	}
	etag := closed.Header().Get("ETag")
	if !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) || len(etag) < 10 {
		t.Fatalf("ETag = %q", etag)
	}

	current := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-10-01&end=2026-10-18")
	if got := current.Header().Get("Cache-Control"); got != "private, max-age=30" {
		t.Errorf("current period Cache-Control = %q", got)
	}
	future := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-10-01&end=2026-12-31")
	if got := future.Header().Get("Cache-Control"); got != "private, max-age=30" {
		t.Errorf("open-ended period Cache-Control = %q", got)
	}

	// Revalidation: same ETag for the same body, 304 without body on a match.
	again := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-07-01&end=2026-09-30")
	if again.Header().Get("ETag") != etag {
		t.Errorf("ETag changed between identical responses")
	}
	for _, inm := range []string{etag, "W/" + etag, `"other", ` + etag, "*"} {
		rec := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-07-01&end=2026-09-30", "If-None-Match", inm)
		wantStatus(t, rec, http.StatusNotModified)
		if rec.Body.Len() != 0 {
			t.Errorf("If-None-Match %q: 304 with a body", inm)
		}
		if rec.Header().Get("ETag") != etag || rec.Header().Get("Cache-Control") == "" {
			t.Errorf("304 must repeat ETag and Cache-Control: %v", rec.Header())
		}
	}
	rec := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-07-01&end=2026-09-30", "If-None-Match", `"stale"`)
	wantStatus(t, rec, http.StatusOK)

	// Different data, different ETag.
	other := serve(h.HeatmapHandler, "/api/stats/heatmap?start=2026-10-01&end=2026-10-18")
	if other.Header().Get("ETag") == etag {
		t.Errorf("different bodies share an ETag")
	}

	// Quarterly is keyed by year: a past year is closed, the current one is not.
	if got := serve(h.QuarterlyHandler, "/api/stats/quarterly?start_year=2024&end_year=2025").Header().Get("Cache-Control"); got != "private, max-age=3600" {
		t.Errorf("closed quarterly Cache-Control = %q", got)
	}
	if got := serve(h.QuarterlyHandler, "/api/stats/quarterly?start_year=2025&end_year=2026").Header().Get("Cache-Control"); got != "private, max-age=30" {
		t.Errorf("current quarterly Cache-Control = %q", got)
	}
	if got := serve(h.GivenHandler, "/api/given?user=U1&day=2026-10-01").Header().Get("Cache-Control"); got != "private, max-age=3600" {
		t.Errorf("closed given Cache-Control = %q", got)
	}
}

type failingWriterValue struct{}

func (failingWriterValue) MarshalJSON() ([]byte, error) { return nil, errors.New("boom") }

func TestWriteJSONEncodingErrorBecomes500(t *testing.T) {
	h := &APIHandlers{logger: zerolog.Nop()}
	rec := httptest.NewRecorder()
	h.respondJSON(rec, httptest.NewRequest(http.MethodGet, "/", nil), "test", failingWriterValue{}, time.Hour)
	wantStatus(t, rec, http.StatusInternalServerError)
	if rec.Header().Get("ETag") != "" {
		t.Errorf("ETag set on error response")
	}
}

func TestRunParallel(t *testing.T) {
	var ran atomic.Int32
	ok := func() error { ran.Add(1); return nil }
	failA := errors.New("a failed")
	failB := errors.New("b failed")

	if err := runParallel([]namedTask{{"one", ok}, {"two", ok}, {"three", ok}}); err != nil || ran.Load() != 3 {
		t.Fatalf("err = %v, ran = %d", err, ran.Load())
	}

	ran.Store(0)
	err := runParallel([]namedTask{
		{"one", ok},
		{"two", func() error { ran.Add(1); return failA }},
		{"three", func() error { ran.Add(1); return failB }},
		{"four", ok},
	})
	if !errors.Is(err, failA) {
		t.Fatalf("err = %v, want the first failing task in list order", err)
	}
	if !strings.HasPrefix(err.Error(), "two: ") {
		t.Errorf("error does not name the task: %q", err)
	}
	if ran.Load() != 4 {
		t.Errorf("ran = %d, want all 4 tasks to finish", ran.Load())
	}

	if err := runParallel(nil); err != nil {
		t.Errorf("no tasks: %v", err)
	}
}

func TestEtagMatches(t *testing.T) {
	const etag = `"abc"`
	for header, want := range map[string]bool{
		"":               false,
		`"abc"`:          true,
		`W/"abc"`:        true,
		`"x","abc"`:      true,
		` "x" ,  "abc" `: true,
		"*":              true,
		`"abcd"`:         false,
		`abc`:            false,
		`"x", W/"y"`:     false,
	} {
		if got := etagMatches(header, etag); got != want {
			t.Errorf("etagMatches(%q) = %v, want %v", header, got, want)
		}
	}
}
