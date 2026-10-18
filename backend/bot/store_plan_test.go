package main

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/BeerBILabs/BeerBI/bot/internal/seed"
)

// planQueries lists every date-bounded query on beers with sample arguments.
func planQueries() map[string]struct {
	sql  string
	args []any
} {
	lo, hi := dateBounds(day("2025-01-01"), day("2025-12-31"))
	ylo, yhi := yearBounds(2020, 2025)
	q := map[string]struct {
		sql  string
		args []any
	}{
		"CountGivenInDateRange":    {qCountGiven, []any{seed.UserID(0), lo, hi}},
		"CountReceivedInDateRange": {qCountReceived, []any{seed.UserID(0), lo, hi}},
		"GetQuarterlyStats":        {qQuarterly, []any{ylo, yhi}},
		"GetHeatmapStats":          {qHeatmap, []any{lo, hi}},
		"GetPairStats":             {qPairs, []any{lo, hi, 15}},
		"GetTopUsers givers":       {topUsersQuery("giver_id"), []any{lo, hi, 20}},
		"GetTopUsers recipients":   {topUsersQuery("recipient_id"), []any{lo, hi, 20}},
	}
	for _, g := range []string{"day", "week", "month"} {
		q["GetTimelineStats "+g] = struct {
			sql  string
			args []any
		}{timelineQuery(g), []any{lo, hi}}
	}
	return q
}

func explain(t *testing.T, s *SQLiteStore, query string, args []any) []string {
	t.Helper()
	rows, err := s.db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var details []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatal(err)
		}
		details = append(details, detail)
	}
	return details
}

// assertSearchPlans guards B1: every date-bounded query must SEARCH an index
// on beers and never SCAN the table or a whole index. With covering set, the
// index must also cover the query (no table lookups).
func assertSearchPlans(t *testing.T, s *SQLiteStore, covering bool) {
	t.Helper()
	for name, q := range planQueries() {
		plan := explain(t, s, q.sql, q.args)
		joined := strings.Join(plan, " | ")
		t.Logf("%s: %s", name, joined)
		if !strings.Contains(joined, "SEARCH beers") || strings.Contains(joined, "SCAN beers") {
			t.Errorf("%s: plan is not an index search on beers: %s", name, joined)
		}
		if covering && !strings.Contains(joined, "SEARCH beers USING COVERING INDEX") {
			t.Errorf("%s: plan reads the table: %s", name, joined)
		}
	}
}

func TestQueryPlans_FreshDB(t *testing.T) {
	assertSearchPlans(t, newTestStore(t), false)
}

// With statistics from a realistic data set the planner must still prefer the
// range search. The statistics come from the production path: no stats yet,
// so the next start (NewSQLiteStore) runs ANALYZE with analysis_limit.
func TestQueryPlans_SeededAndAnalyzed(t *testing.T) {
	s := newTestStore(t)
	if _, err := seed.Generate(context.Background(), s.db, seed.Config{Rows: 20_000, Seed: 7}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM sqlite_stat1`); err != nil {
		t.Fatal(err)
	}
	s, err := NewSQLiteStore(s.db)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if n := statRows(t, s, "idx_beers_ts_count"); n < 10_000 {
		t.Errorf("idx_beers_ts_count stat rows = %d, want the seeded size", n)
	}
	assertSearchPlans(t, s, true)
}

// Regression: statistics taken while the table was small must not freeze.
// The next start (PRAGMA optimize=0x10002 in NewSQLiteStore) refreshes them
// once the table has grown.
func TestQueryPlans_StatsRefreshAfterGrowth(t *testing.T) {
	s := newTestStore(t)
	if _, err := seed.Generate(context.Background(), s.db, seed.Config{Rows: 50, Users: 20, Seed: 3}); err != nil {
		t.Fatalf("seed small: %v", err)
	}
	if _, err := s.db.Exec(`ANALYZE`); err != nil {
		t.Fatal(err)
	}
	if n := statRows(t, s, "idx_beers_ts_count"); n > 100 {
		t.Fatalf("small stat rows = %d, want about 50", n)
	}
	if _, err := seed.Generate(context.Background(), s.db, seed.Config{Rows: 20_000, Seed: 4, Start: day("2018-01-01"), End: day("2020-01-01")}); err != nil {
		t.Fatalf("seed large: %v", err)
	}
	s, err := NewSQLiteStore(s.db)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if n := statRows(t, s, "idx_beers_ts_count"); n < 10_000 {
		t.Errorf("stat rows after growth = %d, want the grown size (stale statistics)", n)
	}
	assertSearchPlans(t, s, true)
}

// statRows returns the row count sqlite_stat1 records for index idx.
func statRows(t *testing.T, s *SQLiteStore, idx string) int {
	t.Helper()
	var stat string
	if err := s.db.QueryRow(`SELECT stat FROM sqlite_stat1 WHERE idx = ?`, idx).Scan(&stat); err != nil {
		t.Fatalf("read sqlite_stat1 for %s: %v", idx, err)
	}
	fields := strings.Fields(stat)
	if len(fields) == 0 {
		t.Fatalf("empty sqlite_stat1 entry for %s", idx)
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		t.Fatalf("parse stat %q: %v", stat, err)
	}
	return n
}
