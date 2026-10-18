package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BeerBILabs/BeerBI/bot/internal/seed"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// captureLog redirects the global zerolog logger (used by migrate) into a
// buffer for the duration of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := zlog.Logger
	zlog.Logger = zerolog.New(&buf)
	t.Cleanup(func() { zlog.Logger = prev })
	return &buf
}

func indexNames(t *testing.T, s *SQLiteStore) map[string]bool {
	t.Helper()
	rows, err := s.db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_autoindex%'`)
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[n] = true
	}
	return out
}

func schemaSnapshot(t *testing.T, s *SQLiteStore) string {
	t.Helper()
	rows, err := s.db.Query(`SELECT type, name, COALESCE(sql, '') FROM sqlite_master WHERE name NOT LIKE 'sqlite_stat%' ORDER BY type, name`)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var sb strings.Builder
	for rows.Next() {
		var typ, name, sql string
		if err := rows.Scan(&typ, &name, &sql); err != nil {
			t.Fatal(err)
		}
		sb.WriteString(typ + " " + name + " " + sql + "\n")
	}
	return sb.String()
}

func userVersion(t *testing.T, s *SQLiteStore) int {
	t.Helper()
	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func execAll(t *testing.T, s *SQLiteStore, stmts ...string) {
	t.Helper()
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			t.Fatalf("exec %q: %v", st, err)
		}
	}
}

func TestMigrate_FreshDBHasAllIndexes(t *testing.T) {
	s := newTestStore(t)

	got := indexNames(t, s)
	for _, idx := range storeIndexes {
		if !got[idx.name] {
			t.Errorf("index %s missing after first migrate; have %v", idx.name, got)
		}
	}
	for _, name := range obsoleteIndexes {
		if got[name] {
			t.Errorf("obsolete index %s exists", name)
		}
	}
	if v := userVersion(t, s); v != schemaVersion {
		t.Errorf("user_version = %d, want %d", v, schemaVersion)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	s := newTestStore(t)
	if err := s.AddBeer("U1", "U2", "1700000000.000001", time.Unix(1700000000, 0), 2); err != nil {
		t.Fatal(err)
	}
	before := schemaSnapshot(t, s)
	for i := 0; i < 2; i++ {
		if err := s.migrate(); err != nil {
			t.Fatalf("migrate run %d: %v", i+2, err)
		}
	}
	if after := schemaSnapshot(t, s); after != before {
		t.Errorf("schema changed on re-run:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if n, _ := s.CountGivenInDateRange("U1", time.Unix(1700000000, 0).UTC(), time.Unix(1700000000, 0).UTC()); n != 2 {
		t.Errorf("count after re-migrate = %d, want 2", n)
	}
}

// A legacy row with neither ts_rfc nor a numeric Slack ts must not abort the
// rebuild (ts_rfc is NOT NULL) and must not be dropped; it gets the explicit
// fallback instead of a value derived from CAST('...' AS INTEGER).
func TestMigrate_RebuildLegacyNonNumericTs(t *testing.T) {
	db, err := openTestDB(filepath.Join(t.TempDir(), "legacy-ts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, st := range []string{
		`CREATE TABLE beers (id INTEGER PRIMARY KEY AUTOINCREMENT, giver_id TEXT NOT NULL, recipient_id TEXT NOT NULL, ts TEXT NOT NULL)`,
		`INSERT INTO beers (giver_id, recipient_id, ts) VALUES ('U1', 'U2', 'not-a-ts')`,
		`INSERT INTO beers (giver_id, recipient_id, ts) VALUES ('U1', 'U2', '1704067200.000100')`,
	} {
		if _, err := db.Exec(st); err != nil {
			t.Fatalf("legacy setup %q: %v", st, err)
		}
	}
	if _, err := NewSQLiteStore(db); err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}
	got := map[string]string{}
	rows, err := db.Query(`SELECT ts, CAST(ts_rfc AS TEXT) FROM beers`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var ts, rfc string
		if err := rows.Scan(&ts, &rfc); err != nil {
			t.Fatal(err)
		}
		got[ts] = rfc
	}
	want := map[string]string{"not-a-ts": legacyFallbackTs, "1704067200.000100": "2024-01-01T00:00:00Z"}
	for ts, w := range want {
		if got[ts] != w {
			t.Errorf("ts %q: ts_rfc = %q, want %q", ts, got[ts], w)
		}
	}
}

// A database from before the UNIQUE constraint: one row per beer, no ts_rfc,
// no count. The rebuild must aggregate duplicates and keep every beer.
func TestMigrate_RebuildsLegacyTableKeepingCounts(t *testing.T) {
	db, err := openTestDB(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	legacy := []string{
		`CREATE TABLE beers (id INTEGER PRIMARY KEY AUTOINCREMENT, giver_id TEXT NOT NULL, recipient_id TEXT NOT NULL, ts TEXT NOT NULL)`,
		// 1704067200 = 2024-01-01T00:00:00Z; the same message twice = 2 beers
		`INSERT INTO beers (giver_id, recipient_id, ts) VALUES ('U1', 'U2', '1704067200.000100')`,
		`INSERT INTO beers (giver_id, recipient_id, ts) VALUES ('U1', 'U2', '1704067200.000100')`,
		`INSERT INTO beers (giver_id, recipient_id, ts) VALUES ('U1', 'U3', '1704153599.000200')`, // 2024-01-01T23:59:59Z
		`INSERT INTO beers (giver_id, recipient_id, ts) VALUES ('U2', 'U1', '1704153600.000300')`, // 2024-01-02T00:00:00Z
	}
	for _, st := range legacy {
		if _, err := db.Exec(st); err != nil {
			t.Fatalf("legacy setup %q: %v", st, err)
		}
	}

	s, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}

	var createSQL string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='beers'`).Scan(&createSQL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(createSQL, "UNIQUE") {
		t.Errorf("beers not rebuilt with UNIQUE: %s", createSQL)
	}
	var tables int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name LIKE 'beers_%'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Errorf("%d leftover beers_* tables", tables)
	}

	var rows, beers int
	if err := db.QueryRow(`SELECT COUNT(1), SUM(count) FROM beers`).Scan(&rows, &beers); err != nil {
		t.Fatal(err)
	}
	if rows != 3 || beers != 4 {
		t.Errorf("rows=%d beers=%d, want 3 rows / 4 beers", rows, beers)
	}

	var tsRFC string
	if err := db.QueryRow(`SELECT CAST(ts_rfc AS TEXT) FROM beers WHERE recipient_id='U3'`).Scan(&tsRFC); err != nil {
		t.Fatal(err)
	}
	if tsRFC != "2024-01-01T23:59:59Z" {
		t.Errorf("derived ts_rfc = %q, want 2024-01-01T23:59:59Z", tsRFC)
	}

	jan1 := day("2024-01-01")
	if n, _ := s.CountGivenInDateRange("U1", jan1, jan1); n != 3 {
		t.Errorf("U1 given on 2024-01-01 = %d, want 3", n)
	}
	for _, idx := range storeIndexes {
		if !indexNames(t, s)[idx.name] {
			t.Errorf("index %s missing after rebuild", idx.name)
		}
	}
}

// A database written by older versions: the misnamed index, user_version 0
// and ts_rfc values in mixed formats.
func TestMigrate_NormalizesLegacyTimestamps(t *testing.T) {
	s := newTestStore(t)
	execAll(t, s,
		`PRAGMA user_version = 0`,
		`DROP INDEX idx_beers_ts_count`,
		`DROP INDEX idx_beers_giver_ts`,
		`CREATE INDEX idx_beers_emoji_ts_rfc ON beers (ts_rfc)`,
		`CREATE INDEX idx_beers_giver_id_ts_rfc ON beers (giver_id, ts_rfc)`,
	)
	rows := []struct{ giver, ts, tsRFC, want string }{
		{"U1", "1704067200.000001", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z"},      // canonical, untouched
		{"U2", "1704067201.000001", "2024-01-01 00:00:01", "2024-01-01T00:00:01Z"},       // legacy datetime()
		{"U3", "1704067202.000001", "2024-01-01T00:00:02.5Z", "2024-01-01T00:00:02Z"},    // fraction
		{"U4", "1704067203.000001", "2024-01-01T01:30:00+02:00", "2023-12-31T23:30:00Z"}, // offset → UTC
		{"U5", "1704067204.000001", "not a date", "2024-01-01T00:00:04Z"},                // from Slack ts
		{"U6", "garbage", "garbage", "garbage"},                                          // unfixable, left alone
	}
	for _, r := range rows {
		if _, err := s.db.Exec(`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES (?, 'UX', ?, ?, 1)`, r.giver, r.ts, r.tsRFC); err != nil {
			t.Fatal(err)
		}
	}

	logs := captureLog(t)
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, r := range rows {
		var got string
		if err := s.db.QueryRow(`SELECT CAST(ts_rfc AS TEXT) FROM beers WHERE giver_id = ?`, r.giver).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != r.want {
			t.Errorf("%s: ts_rfc = %q, want %q", r.giver, got, r.want)
		}
	}
	if !strings.Contains(logs.String(), `"rows_changed":4`) {
		t.Errorf("expected rows_changed=4 in log, got:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "left unchanged") {
		t.Errorf("expected a warning about the unfixable row, got:\n%s", logs.String())
	}
	if v := userVersion(t, s); v != schemaVersion {
		t.Errorf("user_version = %d, want %d", v, schemaVersion)
	}
	idx := indexNames(t, s)
	if idx["idx_beers_emoji_ts_rfc"] || idx["idx_beers_giver_id_ts_rfc"] || !idx["idx_beers_ts_count"] || !idx["idx_beers_giver_ts"] {
		t.Errorf("index replacement failed: %v", idx)
	}

	// Re-running the step (e.g. after a restore of user_version) changes nothing.
	logs.Reset()
	execAll(t, s, `PRAGMA user_version = 0`)
	if err := s.migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if !strings.Contains(logs.String(), `"rows_changed":0`) {
		t.Errorf("expected rows_changed=0 on re-run, got:\n%s", logs.String())
	}
}

// Rows in the legacy "YYYY-MM-DD HH:MM:SS" format are still matched by the
// range predicate, also at the day boundaries (in case such rows appear after
// the one-time normalisation).
func TestDateRange_MatchesLegacyFormat(t *testing.T) {
	s := newTestStore(t)
	execAll(t, s,
		`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES ('U1', 'U2', '1', '2024-03-09 23:59:59', 1)`,
		`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES ('U1', 'U2', '2', '2024-03-10 00:00:00', 2)`,
		`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES ('U1', 'U2', '3', '2024-03-10 23:59:59', 4)`,
		`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES ('U1', 'U2', '4', '2024-03-11 00:00:00', 8)`,
	)
	d := day("2024-03-10")
	if n, _ := s.CountGivenInDateRange("U1", d, d); n != 6 {
		t.Errorf("given on 2024-03-10 = %d, want 6", n)
	}
	if n, _ := s.CountReceivedInDateRange("U2", d.AddDate(0, 0, -1), d); n != 7 {
		t.Errorf("received 03-09…03-10 = %d, want 7", n)
	}
}

func TestSeedSchemaMatchesStore(t *testing.T) {
	got := strings.Replace(seed.BeersDDL, "CREATE TABLE IF NOT EXISTS beers", "CREATE TABLE beers", 1)
	if got != beersCreate {
		t.Errorf("internal/seed.BeersDDL drifted from beersCreate:\n%s\nvs\n%s", seed.BeersDDL, beersCreate)
	}
}
