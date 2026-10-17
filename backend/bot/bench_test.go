package main

// Benchmarks for the analytics queries against a seeded database.
//
//	go test -run '^$' -bench . -benchmem ./...
//
// BEERBI_BENCH_ROWS sets the number of seeded beer rows (default 1,000,000;
// 20,000 with -short). The data set is generated once per test binary by
// internal/seed (500 fake users, 2020-01-01 … 2025-12-31) and deleted in
// TestMain.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/BeerBILabs/BeerBI/bot/internal/seed"
)

var bench struct {
	once  sync.Once
	dir   string
	store *SQLiteStore
	err   error
}

func benchRows() int {
	if v, err := strconv.Atoi(os.Getenv("BEERBI_BENCH_ROWS")); err == nil && v > 0 {
		return v
	}
	if testing.Short() {
		return 20_000
	}
	return 1_000_000
}

// benchStore returns the shared seeded store. The schema is created by a first
// NewSQLiteStore, the rows are seeded, and a second NewSQLiteStore simulates
// the next restart of the bot (which is when migrations see existing data).
func benchStore(b *testing.B) *SQLiteStore {
	b.Helper()
	bench.once.Do(func() {
		bench.dir, bench.err = os.MkdirTemp("", "beerbi-bench-*")
		if bench.err != nil {
			return
		}
		path := filepath.Join(bench.dir, "bench.db")
		db, err := openTestDB(path)
		if err != nil {
			bench.err = err
			return
		}
		if _, err := NewSQLiteStore(db); err != nil {
			bench.err = err
			return
		}
		started := time.Now()
		res, err := seed.Generate(context.Background(), db, seed.Config{Rows: benchRows(), Seed: 1})
		if err != nil {
			bench.err = err
			return
		}
		b.Logf("seeded %d rows (%d beers, %d legacy) in %s", res.Rows, res.Beers, res.LegacyRows, time.Since(started).Round(time.Millisecond))
		bench.store, bench.err = NewSQLiteStore(db)
	})
	if bench.err != nil {
		b.Fatalf("bench store: %v", bench.err)
	}
	return bench.store
}

func cleanupBench() {
	if bench.store != nil {
		_ = bench.store.db.Close()
	}
	if bench.dir != "" {
		_ = os.RemoveAll(bench.dir)
	}
}

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

type benchRange struct {
	name       string
	start, end time.Time
}

// Ranges relative to the seeded period 2020-01-01 … 2025-12-31.
var benchRanges = []benchRange{
	{"month", day("2025-12-01"), day("2025-12-31")},
	{"year", day("2025-01-01"), day("2025-12-31")},
	{"all", day("2020-01-01"), day("2025-12-31")},
}

func BenchmarkGetTopUsers(b *testing.B) {
	s := benchStore(b)
	for _, r := range benchRanges {
		b.Run(r.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := s.GetTopUsers(r.start, r.end, 20); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGetTimelineStats(b *testing.B) {
	s := benchStore(b)
	cases := []struct {
		name        string
		r           benchRange
		granularity string
	}{
		{"day-month", benchRanges[0], "day"},
		{"day-year", benchRanges[1], "day"},
		{"week-year", benchRanges[1], "week"},
		{"month-all", benchRanges[2], "month"},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := s.GetTimelineStats(c.r.start, c.r.end, c.granularity); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCountGivenInDateRange covers the daily-limit check that runs on
// every Slack message (single day) and the per-user stats endpoint (year).
// U000000 is the most active giver in the seeded data.
func BenchmarkCountGivenInDateRange(b *testing.B) {
	s := benchStore(b)
	cases := []benchRange{
		{"day", day("2025-12-15"), day("2025-12-15")},
		benchRanges[1],
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := s.CountGivenInDateRange(seed.UserID(0), c.start, c.end); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCountReceivedInDateRange(b *testing.B) {
	s := benchStore(b)
	r := benchRanges[1]
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.CountReceivedInDateRange(seed.UserID(0), r.start, r.end); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetHeatmapStats(b *testing.B) {
	s := benchStore(b)
	r := benchRanges[1]
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.GetHeatmapStats(r.start, r.end); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetPairStats(b *testing.B) {
	s := benchStore(b)
	for _, r := range benchRanges[1:] {
		b.Run(r.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := s.GetPairStats(r.start, r.end, 15); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGetQuarterlyStats(b *testing.B) {
	s := benchStore(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.GetQuarterlyStats(2020, 2025); err != nil {
			b.Fatal(err)
		}
	}
}
