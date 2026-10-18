// Command seed fills a BeerBI SQLite database with deterministic synthetic
// data for local development and load tests.
//
//	go run ./cmd/seed -db data/bot.db -rows 1000000
//
// It only creates the beers table if it is missing; the bot creates indexes
// and runs its migrations on the next start. All user IDs are fake.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BeerBILabs/BeerBI/bot/internal/seed"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "data/bot.db", "SQLite database file (created if missing)")
	rows := flag.Int("rows", 1_000_000, "number of beer rows to insert")
	users := flag.Int("users", 500, "number of distinct fake users")
	years := flag.Int("years", 6, "length of the period in years, ending at -end")
	end := flag.String("end", "2026-01-01", "end of the period (exclusive, YYYY-MM-DD)")
	legacyEvery := flag.Int("legacy-every", 100, "store every n-th row with a legacy 'YYYY-MM-DD HH:MM:SS' ts_rfc (-1 disables)")
	seedVal := flag.Uint64("seed", 1, "random seed")
	flag.Parse()

	endT, err := time.Parse("2006-01-02", *end)
	if err != nil {
		log.Fatalf("invalid -end: %v", err)
	}
	if err := os.MkdirAll(dirOf(*dbPath), 0o755); err != nil {
		log.Fatalf("create directory: %v", err)
	}

	db, err := sql.Open("sqlite", *dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(seed.BeersDDL); err != nil {
		log.Fatalf("create beers table: %v", err)
	}
	// Generate is deterministic: a second run would repeat rows and fail on
	// UNIQUE(giver_id, recipient_id, ts) halfway through.
	var existing int
	if err := db.QueryRow(`SELECT COUNT(1) FROM beers`).Scan(&existing); err != nil {
		log.Fatalf("count beers: %v", err)
	}
	if existing > 0 {
		log.Fatalf("%s already contains %d beers; seed only an empty database (delete the file first)", *dbPath, existing)
	}

	started := time.Now()
	res, err := seed.Generate(context.Background(), db, seed.Config{
		Users:       *users,
		Rows:        *rows,
		Start:       endT.AddDate(-*years, 0, 0),
		End:         endT,
		LegacyEvery: *legacyEvery,
		Seed:        *seedVal,
	})
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	fmt.Printf("inserted %d rows (%d beers, %d legacy ts_rfc) into %s in %s\n",
		res.Rows, res.Beers, res.LegacyRows, *dbPath, time.Since(started).Round(time.Millisecond))
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
