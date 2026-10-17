package main

import (
	"context"
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

// With statistics from a realistic data set (ANALYZE after the migration),
// the planner must still prefer the range search.
func TestQueryPlans_SeededAndAnalyzed(t *testing.T) {
	s := newTestStore(t)
	if _, err := seed.Generate(context.Background(), s.db, seed.Config{Rows: 20_000, Seed: 7}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.db.Exec(`ANALYZE`); err != nil {
		t.Fatal(err)
	}
	assertSearchPlans(t, s, true)
}
