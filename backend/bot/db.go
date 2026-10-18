package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	zlog "github.com/rs/zerolog/log"
	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO), registered as "sqlite"
)

// sqlitePragmas are executed by the driver on every new pool connection (in
// this order after busy_timeout, which the driver always runs first).
var sqlitePragmas = []string{
	// Wait up to 5 s for a lock instead of failing with SQLITE_BUSY.
	"busy_timeout(5000)",
	// WAL: readers never block the writer and the writer never blocks readers
	// (the bot writes while dashboards read). Persistent in the file.
	"journal_mode(WAL)",
	// Safe with WAL (no corruption on power loss, at most the last commits are
	// lost); fsync only at checkpoints instead of on every commit.
	"synchronous(NORMAL)",
	"foreign_keys(1)",
	// 16 MB page cache per connection (default 2 MB), see maxOpenConns.
	"cache_size(-16000)",
	"temp_store(MEMORY)",
}

// maxOpenConns bounds the connection pool. WAL allows any number of readers
// next to one writer; SQLite serialises writers itself. Writes are rare (one
// row per beer message), so instead of a separate single-connection writer
// pool every transaction starts with BEGIN IMMEDIATE (_txlock): a writer takes
// the write lock up front and other writers wait for it in busy_timeout,
// rather than failing with SQLITE_BUSY when a read transaction tries to
// upgrade. Four connections let the analytics queries of one dashboard load
// run in parallel while keeping the page caches within the container's memory
// limit (4 × 16 MB).
const maxOpenConns = 4

// sqliteDSN builds the modernc.org/sqlite DSN for path.
func sqliteDSN(path string) (string, error) {
	if strings.ContainsAny(path, "?#") {
		return "", fmt.Errorf("sqlite path %q must not contain '?' or '#'", path)
	}
	q := url.Values{}
	for _, p := range sqlitePragmas {
		q.Add("_pragma", p)
	}
	q.Set("_txlock", "immediate")
	return path + "?" + q.Encode(), nil
}

// openDB opens the SQLite database at path with the production settings
// (WAL, busy_timeout, synchronous=NORMAL, bounded pool). Tests use it too, so
// they run against the same configuration.
func openDB(path string) (*sql.DB, error) {
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)
	// Connections live for the whole process: each one holds its page cache
	// and prepared state, and SQLite connections do not go stale.
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	// Opening is lazy; connect now so a bad path or pragma fails at startup.
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if !strings.EqualFold(mode, "wal") {
		// In-memory databases cannot use WAL; anything else is unexpected.
		zlog.Warn().Str("journal_mode", mode).Str("path", path).Msg("sqlite: WAL is not active")
	}
	return db, nil
}
