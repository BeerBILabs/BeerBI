package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func TestMain(m *testing.M) {
	// Keep test output readable; tests that check log lines use captureLog.
	zlog.Logger = zerolog.Nop()
	code := m.Run()
	cleanupBench()
	os.Exit(code)
}

// openTestDB opens a SQLite file the same way the tests have always done.
func openTestDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path+"?_foreign_keys=1")
}

// newTestStore returns a migrated store on a fresh file in t.TempDir().
func newTestStore(t testing.TB) *SQLiteStore {
	t.Helper()
	db, err := openTestDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}
