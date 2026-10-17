package seed

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func seededDB(t *testing.T, cfg Config) (*sql.DB, Result) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(BeersDDL); err != nil {
		t.Fatal(err)
	}
	res, err := Generate(context.Background(), db, cfg)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return db, res
}

func digest(t *testing.T, db *sql.DB) string {
	t.Helper()
	var d string
	err := db.QueryRow(`SELECT group_concat(giver_id || recipient_id || ts || CAST(ts_rfc AS TEXT) || count, ',') FROM (SELECT * FROM beers ORDER BY id)`).Scan(&d)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestGenerate(t *testing.T) {
	cfg := Config{Users: 50, Rows: 75_123, LegacyEvery: 10, Seed: 42}
	db, res := seededDB(t, cfg)

	if res.Rows != cfg.Rows || res.LegacyRows != 7_513 || res.Beers < res.Rows {
		t.Errorf("result = %+v", res)
	}
	var rows, beers, self, users int
	var minTs, maxTs string
	err := db.QueryRow(`SELECT COUNT(1), SUM(count), SUM(giver_id = recipient_id),
		(SELECT COUNT(DISTINCT giver_id) FROM beers), MIN(CAST(ts_rfc AS TEXT)), MAX(CAST(ts_rfc AS TEXT)) FROM beers`).
		Scan(&rows, &beers, &self, &users, &minTs, &maxTs)
	if err != nil {
		t.Fatal(err)
	}
	if rows != res.Rows || beers != res.Beers {
		t.Errorf("db has %d rows / %d beers, result says %+v", rows, beers, res)
	}
	if self != 0 {
		t.Errorf("%d self-gifts", self)
	}
	if users < 2 || users > cfg.Users {
		t.Errorf("distinct givers = %d", users)
	}
	if minTs < "2020-01-01" || maxTs >= "2026-01-01" {
		t.Errorf("timestamps out of range: %s … %s", minTs, maxTs)
	}

	var legacy int
	if err := db.QueryRow(`SELECT COUNT(1) FROM beers WHERE CAST(ts_rfc AS TEXT) NOT LIKE '%T%Z'`).Scan(&legacy); err != nil {
		t.Fatal(err)
	}
	if legacy != res.LegacyRows {
		t.Errorf("legacy rows in db = %d, want %d", legacy, res.LegacyRows)
	}
}

func TestGenerateDeterministic(t *testing.T) {
	cfg := Config{Users: 20, Rows: 3_000, Seed: 9}
	a, _ := seededDB(t, cfg)
	b, _ := seededDB(t, cfg)
	if digest(t, a) != digest(t, b) {
		t.Error("same config produced different data")
	}
	cfg.Seed = 10
	c, _ := seededDB(t, cfg)
	if digest(t, a) == digest(t, c) {
		t.Error("different seeds produced identical data")
	}
}

func TestDefaultsAndUserID(t *testing.T) {
	c := Config{}.withDefaults()
	if c.Users != 500 || c.Rows != 1_000_000 || c.LegacyEvery != 100 ||
		!c.Start.Equal(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) || !c.End.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("defaults = %+v", c)
	}
	if id := UserID(123); id != "U000123" {
		t.Errorf("UserID(123) = %s", id)
	}
	if !strings.HasPrefix(insertSQL(2), "INSERT INTO beers") || strings.Count(insertSQL(3), "(?,?,?,?,?)") != 3 {
		t.Errorf("insertSQL = %s", insertSQL(3))
	}
}
