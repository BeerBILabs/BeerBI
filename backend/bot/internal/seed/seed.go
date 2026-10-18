// Package seed fills a BeerBI SQLite database with deterministic synthetic
// beer data. It is used by the benchmarks (bench_test.go) and by the
// cmd/seed tool for local development with realistic data volumes.
//
// All user IDs are fake (U000000 … U000499); no real Slack data is involved.
package seed

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

// BeersDDL is the schema of the beers table. It must match beersCreate in
// store.go (guarded by a test in the bot package). The bot's migrate() adds
// all indexes on its next start.
const BeersDDL = `CREATE TABLE IF NOT EXISTS beers (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            giver_id TEXT NOT NULL,
            recipient_id TEXT NOT NULL,
            ts TEXT NOT NULL, -- original Slack ts string (with fraction)
            ts_rfc DATETIME NOT NULL, -- parsed RFC3339 time for date queries
            count INTEGER NOT NULL DEFAULT 1,
            UNIQUE (giver_id, recipient_id, ts)
        );`

// LegacyLayout is the ts_rfc format written by the old legacy migration path
// (SQLite datetime(), no "T", no "Z").
const LegacyLayout = "2006-01-02 15:04:05"

// Config controls the generated data set.
type Config struct {
	Users int       // number of distinct users (default 500)
	Rows  int       // number of beer rows (default 1,000,000)
	Start time.Time // first instant, inclusive (default 2020-01-01 UTC)
	End   time.Time // last instant, exclusive (default 2026-01-01 UTC, i.e. 6 years)
	// LegacyEvery stores every n-th row with a legacy "YYYY-MM-DD HH:MM:SS"
	// ts_rfc value. 0 uses the default (100), a negative value disables it.
	LegacyEvery int
	Seed        uint64 // RNG seed; the same config always yields the same rows
}

// Result describes what was written.
type Result struct {
	Rows       int // rows inserted
	Beers      int // SUM(count) of the inserted rows
	LegacyRows int // rows with a legacy ts_rfc format
}

func (c Config) withDefaults() Config {
	if c.Users <= 1 {
		c.Users = 500
	}
	if c.Rows <= 0 {
		c.Rows = 1_000_000
	}
	if c.Start.IsZero() {
		c.Start = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if c.End.IsZero() || !c.End.After(c.Start) {
		c.End = c.Start.AddDate(6, 0, 0)
	}
	if c.LegacyEvery == 0 {
		c.LegacyEvery = 100
	}
	return c
}

// UserID returns the fake Slack user ID for index i (U000000, U000001, …).
func UserID(i int) string { return fmt.Sprintf("U%06d", i) }

// Generate inserts cfg.Rows synthetic beers into the existing beers table.
//
// Shape of the data: givers and recipients follow two independent Zipf
// distributions (a few very active users, a long tail), activity grows over
// the years, weekends are quiet, office hours dominate, and most messages give
// one beer (some two or three, rarely five).
func Generate(ctx context.Context, db *sql.DB, cfg Config) (Result, error) {
	cfg = cfg.withDefaults()
	rng := rand.New(rand.NewPCG(cfg.Seed, 0x9e3779b97f4a7c15))

	ids := make([]string, cfg.Users)
	for i := range ids {
		ids[i] = UserID(i)
	}
	// Recipients use a shuffled ranking so the top givers are not automatically
	// the top recipients.
	recipientRank := rng.Perm(cfg.Users)
	giverZipf := rand.NewZipf(rng, 1.15, 2, uint64(cfg.Users-1))
	recipientZipf := rand.NewZipf(rng, 1.05, 3, uint64(cfg.Users-1))

	span := cfg.End.Sub(cfg.Start)
	const batch = 200
	const txRows = 50_000

	var res Result
	tx, stmt, err := beginBatch(ctx, db, batch)
	if err != nil {
		return res, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	args := make([]any, 0, batch*5)
	flush := func(s *sql.Stmt) error {
		if len(args) == 0 {
			return nil
		}
		if s == nil {
			n := len(args) / 5
			q := insertSQL(n)
			if _, err := tx.ExecContext(ctx, q, args...); err != nil {
				return fmt.Errorf("seed insert: %w", err)
			}
		} else if _, err := s.ExecContext(ctx, args...); err != nil {
			return fmt.Errorf("seed insert: %w", err)
		}
		args = args[:0]
		return nil
	}

	for i := 0; i < cfg.Rows; i++ {
		t := randomInstant(rng, cfg.Start, span)
		giver := int(giverZipf.Uint64())
		recipient := recipientRank[int(recipientZipf.Uint64())]
		if recipient == giver {
			recipient = (recipient + 1 + rng.IntN(cfg.Users-1)) % cfg.Users
		}
		count := 1
		switch p := rng.Float64(); {
		case p < 0.01:
			count = 5
		case p < 0.05:
			count = 3
		case p < 0.15:
			count = 2
		}
		// Slack ts: seconds + fraction. The fraction encodes the row number
		// (six digits, plus the millions as extra digits from row 1,000,000
		// on), so (giver, recipient, ts) is unique within one Generate call.
		// A second call with the same Config on the same table repeats the
		// rows and fails on the UNIQUE constraint; cmd/seed refuses non-empty
		// tables for that reason.
		slackTs := fmt.Sprintf("%d.%06d", t.Unix(), i%1_000_000)
		if i >= 1_000_000 {
			slackTs += strconv.Itoa(i / 1_000_000)
		}
		tsRFC := t.UTC().Format(time.RFC3339)
		if cfg.LegacyEvery > 0 && i%cfg.LegacyEvery == 0 {
			tsRFC = t.UTC().Format(LegacyLayout)
			res.LegacyRows++
		}
		args = append(args, ids[giver], ids[recipient], slackTs, tsRFC, count)
		res.Rows++
		res.Beers += count

		if len(args) == batch*5 {
			if err := flush(stmt); err != nil {
				return res, err
			}
		}
		if (i+1)%txRows == 0 {
			if err := tx.Commit(); err != nil {
				return res, fmt.Errorf("seed commit: %w", err)
			}
			tx, stmt, err = beginBatch(ctx, db, batch)
			if err != nil {
				return res, err
			}
		}
	}
	if err := flush(nil); err != nil {
		return res, err
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("seed commit: %w", err)
	}
	tx = nil
	return res, nil
}

func beginBatch(ctx context.Context, db *sql.DB, n int) (*sql.Tx, *sql.Stmt, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("seed begin: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, insertSQL(n))
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("seed prepare: %w", err)
	}
	return tx, stmt, nil
}

func insertSQL(n int) string {
	var sb strings.Builder
	sb.WriteString("INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES ")
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?,?)")
	}
	return sb.String()
}

// randomInstant draws a time in [start, start+span) with a growth trend over
// the years, quiet weekends and most activity between 08:00 and 19:00 UTC.
func randomInstant(rng *rand.Rand, start time.Time, span time.Duration) time.Time {
	for {
		// Linear growth: density rises from 1 to 3 across the span
		// (inverse-CDF sampling of f(x) = 1 + 2x on [0,1)).
		u := rng.Float64()
		x := (-1 + math.Sqrt(1+8*u)) / 2
		t := start.Add(time.Duration(x * float64(span)))
		if wd := t.Weekday(); (wd == time.Saturday || wd == time.Sunday) && rng.Float64() < 0.9 {
			continue
		}
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		var sec int
		if rng.Float64() < 0.85 {
			sec = 8*3600 + rng.IntN(11*3600)
		} else {
			sec = rng.IntN(24 * 3600)
		}
		out := day.Add(time.Duration(sec) * time.Second)
		if out.Before(start) || !out.Before(start.Add(span)) {
			continue
		}
		return out
	}
}
