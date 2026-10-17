package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	zlog "github.com/rs/zerolog/log"
)

// dateLayout is the calendar-day format used for query bounds.
const dateLayout = "2006-01-02"

// schemaVersion is stored in PRAGMA user_version. Version 1 normalises all
// beers.ts_rfc values to RFC3339 UTC (see normalizeTimestamps).
const schemaVersion = 1

// rfc3339UTCGlob matches the canonical ts_rfc format written by AddBeer.
const rfc3339UTCGlob = "[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]Z"

// beersCreate is the schema of the beers table (keep in sync with
// internal/seed.BeersDDL, guarded by TestSeedSchemaMatchesStore).
const beersCreate = `CREATE TABLE beers (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            giver_id TEXT NOT NULL,
            recipient_id TEXT NOT NULL,
            ts TEXT NOT NULL, -- original Slack ts string (with fraction)
            ts_rfc DATETIME NOT NULL, -- parsed RFC3339 time for date queries
            count INTEGER NOT NULL DEFAULT 1,
            UNIQUE (giver_id, recipient_id, ts)
        );`

// storeIndexes are created at the end of every migration (IF NOT EXISTS).
// All three are covering indexes for the queries that use them, so date-range
// queries never read the table itself (measured in bench_test.go):
//
//   - idx_beers_giver_ts: per-giver counts (daily-limit check, /api/given),
//     top givers and giver→recipient pairs. With ANALYZE statistics SQLite
//     uses it as a skip-scan over the few hundred givers.
//   - idx_beers_recipient_ts: per-recipient counts and top recipients.
//   - idx_beers_ts_count: timeline, heatmap and quarterly aggregates;
//     recipient_id lets it also cover top recipients when the planner prefers
//     a range scan (small tables).
var storeIndexes = []struct{ name, ddl string }{
	{"idx_beers_giver_ts", `CREATE INDEX IF NOT EXISTS idx_beers_giver_ts ON beers (giver_id, ts_rfc, recipient_id, count)`},
	{"idx_beers_recipient_ts", `CREATE INDEX IF NOT EXISTS idx_beers_recipient_ts ON beers (recipient_id, ts_rfc, count)`},
	{"idx_beers_ts_count", `CREATE INDEX IF NOT EXISTS idx_beers_ts_count ON beers (ts_rfc, recipient_id, count)`},
	{"idx_emoji_counts_user_id_emoji", `CREATE INDEX IF NOT EXISTS idx_emoji_counts_user_id_emoji ON emoji_counts (user_id, emoji)`},
}

// obsoleteIndexes are dropped during migration: the misnamed
// idx_beers_emoji_ts_rfc on (ts_rfc) and the non-covering per-user indexes,
// all replaced by the covering indexes above.
var obsoleteIndexes = []string{"idx_beers_emoji_ts_rfc", "idx_beers_giver_id_ts_rfc", "idx_beers_recipient_id_ts_rfc"}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// migrate brings the schema up to date. Every step is idempotent, so it runs
// on every start:
//
//  1. auxiliary tables
//  2. beers table: create, add missing columns, or rebuild a legacy table
//     without the UNIQUE constraint
//  3. drop obsolete indexes
//  4. versioned data migrations (PRAGMA user_version)
//  5. create indexes (after the table exists, so a fresh DB gets all of them)
//  6. ANALYZE when indexes were created or statistics are missing
func (s *SQLiteStore) migrate() error {
	aux := []string{
		`CREATE TABLE IF NOT EXISTS emoji_counts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			emoji TEXT NOT NULL,
			count INTEGER NOT NULL DEFAULT 0,
			UNIQUE(user_id, emoji)
		);`,
		`CREATE TABLE IF NOT EXISTS processed_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT NOT NULL UNIQUE,
			ts TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS user_cache (
			user_id TEXT PRIMARY KEY,
			real_name TEXT NOT NULL,
			profile_image TEXT,
			updated_at DATETIME NOT NULL
		);`,
	}
	for _, st := range aux {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("migrate exec: %w", err)
		}
	}

	if err := s.migrateBeersTable(); err != nil {
		return err
	}

	for _, name := range obsoleteIndexes {
		if _, err := s.db.Exec(`DROP INDEX IF EXISTS ` + name); err != nil {
			return fmt.Errorf("migrate drop index %s: %w", name, err)
		}
	}

	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("migrate read user_version: %w", err)
	}
	if version < 1 {
		if err := s.normalizeTimestamps(); err != nil {
			return err
		}
	}

	created := 0
	for _, idx := range storeIndexes {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='index' AND name=?`, idx.name).Scan(&n); err != nil {
			return fmt.Errorf("migrate check index %s: %w", idx.name, err)
		}
		if n > 0 {
			continue
		}
		started := time.Now()
		if _, err := s.db.Exec(idx.ddl); err != nil {
			return fmt.Errorf("migrate create index %s: %w", idx.name, err)
		}
		created++
		zlog.Info().Str("index", idx.name).Dur("took", time.Since(started)).Msg("migrate: created index")
	}

	return s.analyzeIfNeeded(created > 0)
}

// migrateBeersTable creates the beers table or upgrades an existing one.
func (s *SQLiteStore) migrateBeersTable() error {
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='beers'`).Scan(&exists); err != nil {
		return fmt.Errorf("migrate check beers exists: %w", err)
	}
	if exists == 0 {
		if _, err := s.db.Exec(beersCreate); err != nil {
			return fmt.Errorf("migrate create beers: %w", err)
		}
		return nil
	}

	cols, err := s.tableColumns("beers")
	if err != nil {
		return err
	}
	// Add missing columns non-destructively.
	if !cols["ts_rfc"] {
		if _, err := s.db.Exec(`ALTER TABLE beers ADD COLUMN ts_rfc DATETIME;`); err != nil {
			return fmt.Errorf("migrate add ts_rfc: %w", err)
		}
	}
	if !cols["count"] {
		if _, err := s.db.Exec(`ALTER TABLE beers ADD COLUMN count INTEGER NOT NULL DEFAULT 1;`); err != nil {
			return fmt.Errorf("migrate add count: %w", err)
		}
	}

	// Ensure UNIQUE(giver_id, recipient_id, ts) exists. SQLite cannot add a
	// UNIQUE constraint via ALTER, so a legacy table is rebuilt: the old rows
	// are aggregated per (giver, recipient, ts) into a new table, in one
	// transaction.
	var createSQL sql.NullString
	if err := s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='beers'`).Scan(&createSQL); err != nil {
		return fmt.Errorf("migrate select create sql: %w", err)
	}
	if createSQL.Valid && strings.Contains(strings.ToUpper(createSQL.String), "UNIQUE") {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("migrate begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	steps := []struct{ what, sql string }{
		{"rename legacy beers", `ALTER TABLE beers RENAME TO beers_legacy`},
		{"create beers", beersCreate},
		// Missing ts_rfc is derived from the Slack ts (seconds before the dot).
		{"copy aggregated", `INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count)
			SELECT giver_id, recipient_id, ts,
				COALESCE(MIN(ts_rfc), strftime('%Y-%m-%dT%H:%M:%SZ', CAST(ts AS INTEGER), 'unixepoch')),
				COALESCE(SUM(count), COUNT(1))
			FROM beers_legacy GROUP BY giver_id, recipient_id, ts`},
		{"drop legacy beers", `DROP TABLE beers_legacy`},
	}
	for _, st := range steps {
		if _, err := tx.Exec(st.sql); err != nil {
			return fmt.Errorf("migrate rebuild beers (%s): %w", st.what, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate commit rebuild: %w", err)
	}
	zlog.Info().Msg("migrate: rebuilt legacy beers table with UNIQUE(giver_id, recipient_id, ts)")
	return nil
}

func (s *SQLiteStore) tableColumns(table string) (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, fmt.Errorf("migrate table_info %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	cols := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("migrate scan table_info: %w", err)
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// normalizeTimestamps rewrites every beers.ts_rfc value that is not
// canonical RFC3339 UTC ("YYYY-MM-DDTHH:MM:SSZ"), e.g. the legacy
// "YYYY-MM-DD HH:MM:SS" written by the old rebuild path, values with a
// fraction or an offset, or NULL. The value is re-derived from ts_rfc itself
// (SQLite converts offsets to UTC) or, if that is unparseable, from the Slack
// ts. Rows where neither works are left untouched and reported.
//
// It runs in one transaction together with the user_version bump, so it is
// all-or-nothing and runs once; running it again changes nothing.
func (s *SQLiteStore) normalizeTimestamps() error {
	const canonical = `COALESCE(
			CASE WHEN typeof(ts_rfc) = 'text' THEN strftime('%Y-%m-%dT%H:%M:%SZ', ts_rfc) END,
			CASE WHEN ts GLOB '[0-9]*' THEN strftime('%Y-%m-%dT%H:%M:%SZ', CAST(ts AS INTEGER), 'unixepoch') END)`
	const notCanonical = `(ts_rfc IS NULL OR typeof(ts_rfc) <> 'text' OR ts_rfc NOT GLOB '` + rfc3339UTCGlob + `')`

	started := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("migrate normalize begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`UPDATE beers SET ts_rfc = ` + canonical + ` WHERE ` + notCanonical + ` AND ` + canonical + ` IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("migrate normalize ts_rfc: %w", err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("migrate normalize rows affected: %w", err)
	}
	var remaining int64
	if err := tx.QueryRow(`SELECT COUNT(1) FROM beers WHERE ` + notCanonical).Scan(&remaining); err != nil {
		return fmt.Errorf("migrate normalize count remaining: %w", err)
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("migrate set user_version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate normalize commit: %w", err)
	}

	zlog.Info().Int64("rows_changed", changed).Dur("took", time.Since(started)).Msg("migrate: normalised beers.ts_rfc to RFC3339 UTC")
	if remaining > 0 {
		zlog.Warn().Int64("rows", remaining).Msg("migrate: beers rows with an unparseable ts_rfc and ts were left unchanged")
	}
	return nil
}

// analyzeIfNeeded refreshes the planner statistics when indexes were just
// created or no statistics exist yet for beers. analysis_limit keeps ANALYZE
// fast on large tables (it samples instead of reading every index entry).
func (s *SQLiteStore) analyzeIfNeeded(force bool) error {
	if !force {
		var n int
		err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='sqlite_stat1'`).Scan(&n)
		if err != nil {
			return fmt.Errorf("migrate check stat1: %w", err)
		}
		if n > 0 {
			if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_stat1 WHERE tbl='beers'`).Scan(&n); err != nil {
				return fmt.Errorf("migrate read stat1: %w", err)
			}
		}
		if n > 0 {
			return nil
		}
	}
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate analyze conn: %w", err)
	}
	defer func() { _ = conn.Close() }()
	started := time.Now()
	if _, err := conn.ExecContext(ctx, `PRAGMA analysis_limit = 1000`); err != nil {
		return fmt.Errorf("migrate analysis_limit: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `ANALYZE`); err != nil {
		return fmt.Errorf("migrate analyze: %w", err)
	}
	zlog.Info().Dur("took", time.Since(started)).Msg("migrate: analyzed database")
	return nil
}

// dateBounds converts the inclusive calendar-day range start … end into the
// half-open string range [lo, hi) used as `ts_rfc >= lo AND ts_rfc < hi`.
//
// lo is the start day "YYYY-MM-DD", hi is the day after end. Both are bare
// dates, never "…T00:00:00Z": every ts_rfc of day D starts with "D" and is
// longer than it, so it sorts after "D" and before "D+1". This keeps the
// predicate sargable (index SEARCH) and still matches any ts_rfc format that
// starts with the date, including the legacy "YYYY-MM-DD HH:MM:SS".
// The calendar day is taken in each time's own location, as before.
func dateBounds(start, end time.Time) (lo, hi string) {
	y, m, d := end.Date()
	return start.Format(dateLayout), time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC).Format(dateLayout)
}

// yearBounds returns the [lo, hi) ts_rfc range for the years startYear …
// endYear (inclusive). Years are clamped to 0 … 9998 so the bounds stay
// four-digit strings that compare correctly.
func yearBounds(startYear, endYear int) (lo, hi string) {
	startYear = min(max(startYear, 0), 9998)
	endYear = min(max(endYear, -1), 9998)
	return fmt.Sprintf("%04d-01-01", startYear), fmt.Sprintf("%04d-01-01", endYear+1)
}

// MarkEventProcessed records that an external event (by event_id) has been
// handled. Returns nil if inserted; if the event already exists, returns nil as well.
func (s *SQLiteStore) MarkEventProcessed(eventID string, ts time.Time) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO processed_events (event_id, ts) VALUES (?, ?);`, eventID, ts.UTC().Format(time.RFC3339))
	return err
}

// TryMarkEventProcessed attempts to insert the event id into processed_events.
// Returns (true, nil) if we recorded the event (i.e. this process should handle it),
// (false, nil) if the event was already present (another process handled it),
// or (false, err) on database error.
func (s *SQLiteStore) TryMarkEventProcessed(eventID string, ts time.Time) (bool, error) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO processed_events (event_id, ts) VALUES (?, ?);`, eventID, ts.UTC().Format(time.RFC3339))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// IsEventProcessed returns true if we've already processed the given event id.
func (s *SQLiteStore) IsEventProcessed(eventID string) (bool, error) {
	var id int
	err := s.db.QueryRow(`SELECT id FROM processed_events WHERE event_id = ?`, eventID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// PruneProcessedEvents deletes deduplication records older than cutoff and
// returns how many were removed. Slack redelivers events within minutes, so
// old records only make the UNIQUE index grow.
func (s *SQLiteStore) PruneProcessedEvents(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM processed_events WHERE ts < ?`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("prune processed_events: %w", err)
	}
	return res.RowsAffected()
}

// RunProcessedEventsPruner prunes processed_events records older than maxAge
// right away and then every interval, until ctx is cancelled. Run it in its
// own goroutine; it returns when ctx is done.
func (s *SQLiteStore) RunProcessedEventsPruner(ctx context.Context, interval, maxAge time.Duration) {
	prune := func() {
		n, err := s.PruneProcessedEvents(time.Now().Add(-maxAge))
		if err != nil {
			zlog.Error().Err(err).Msg("processed_events prune failed")
			return
		}
		zlog.Debug().Int64("deleted", n).Msg("processed_events pruned")
	}
	prune()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prune()
		}
	}
}

func (s *SQLiteStore) IncEmoji(userID, emoji string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// try update
	res, err := tx.Exec(`UPDATE emoji_counts SET count = count + 1 WHERE user_id = ? AND emoji = ?`, userID, emoji)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := tx.Exec(`INSERT INTO emoji_counts(user_id, emoji, count) VALUES(?, ?, 1)`, userID, emoji); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetCount(userID, emoji string) (int, error) {
	var c int
	err := s.db.QueryRow(`SELECT count FROM emoji_counts WHERE user_id = ? AND emoji = ?`, userID, emoji).Scan(&c)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return c, nil
}

// AddBeer records a beer-gift event for a single message: it inserts or upserts
// a row with the provided count keyed by the original Slack ts string (ts).
// If the same (giver, recipient, ts) already exists, the count is updated to
// the provided value (last write wins). ts_rfc is stored as RFC3339 UTC.
func (s *SQLiteStore) AddBeer(giverID, recipientID string, slackTs string, t time.Time, count int) error {
	_, err := s.db.Exec(`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES (?, ?, ?, ?, ?) ON CONFLICT(giver_id, recipient_id, ts) DO UPDATE SET count = excluded.count`, giverID, recipientID, slackTs, t.UTC().Format(time.RFC3339), count)
	return err
}

// CountGivenInDateRange returns how many beers the giver gave in the given
// date range (calendar days, both ends inclusive).
func (s *SQLiteStore) CountGivenInDateRange(giverID string, start time.Time, end time.Time) (int, error) {
	lo, hi := dateBounds(start, end)
	var c int
	err := s.db.QueryRow(qCountGiven, giverID, lo, hi).Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}

// CountReceivedInDateRange returns total beers received by recipient in the
// given date range (calendar days, both ends inclusive).
func (s *SQLiteStore) CountReceivedInDateRange(recipientID string, start time.Time, end time.Time) (int, error) {
	lo, hi := dateBounds(start, end)
	var c int
	err := s.db.QueryRow(qCountReceived, recipientID, lo, hi).Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}

// CountGivenOnDate returns how many beers the giver gave on the given date (YYYY-MM-DD)
func (s *SQLiteStore) CountGivenOnDate(giverID string, date string) (int, error) {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0, err
	}
	return s.CountGivenInDateRange(giverID, t, t)
}

// CountReceived returns total beers received by recipient (optionally filtered by date if not empty)
func (s *SQLiteStore) CountReceived(recipientID string, date string) (int, error) {
	if date == "" {
		return s.CountReceivedInDateRange(recipientID, time.Time{}, time.Now())
	}
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0, err
	}
	return s.CountReceivedInDateRange(recipientID, t, t)
}

// GetAllGivers returns the list of all distinct user IDs that have given at least one beer.
func (s *SQLiteStore) GetAllGivers() ([]string, error) {
	return s.queryStrings(`SELECT DISTINCT giver_id FROM beers`)
}

// GetAllRecipients returns the list of all distinct recipient user IDs that have received at least one beer.
func (s *SQLiteStore) GetAllRecipients() ([]string, error) {
	return s.queryStrings(`SELECT DISTINCT recipient_id FROM beers`)
}

func (s *SQLiteStore) queryStrings(query string, args ...any) ([]string, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// CachedUser represents a cached Slack user
type CachedUser struct {
	UserID       string
	RealName     string
	ProfileImage string
	UpdatedAt    time.Time
}

// GetCachedUser retrieves a user from the cache by user ID
func (s *SQLiteStore) GetCachedUser(userID string) (*CachedUser, error) {
	var u CachedUser
	var profileImage sql.NullString
	// updated_at is declared DATETIME, so the driver may already return a
	// time.Time instead of the stored RFC3339 string.
	var updatedAt any
	err := s.db.QueryRow(`SELECT user_id, real_name, profile_image, updated_at FROM user_cache WHERE user_id = ?`, userID).Scan(&u.UserID, &u.RealName, &profileImage, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if profileImage.Valid {
		u.ProfileImage = profileImage.String
	}
	switch v := updatedAt.(type) {
	case time.Time:
		u.UpdatedAt = v.UTC()
	case string:
		u.UpdatedAt, _ = time.Parse(time.RFC3339, v)
	case []byte:
		u.UpdatedAt, _ = time.Parse(time.RFC3339, string(v))
	}
	return &u, nil
}

// SetCachedUser stores or updates a user in the cache
func (s *SQLiteStore) SetCachedUser(userID, realName, profileImage string) error {
	_, err := s.db.Exec(`INSERT INTO user_cache (user_id, real_name, profile_image, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET real_name = excluded.real_name, profile_image = excluded.profile_image, updated_at = excluded.updated_at`,
		userID, realName, profileImage, time.Now().UTC().Format(time.RFC3339))
	return err
}

// ============================================================================
// Stats Query Methods for Analytics/BI Features
//
// Every date filter is the sargable range `ts_rfc >= ? AND ts_rfc < ?` built by
// dateBounds/yearBounds, so SQLite searches a covering index instead of
// scanning the table (guarded by store_plan_test.go).
// ============================================================================

// TimelinePoint represents a single data point in a timeline chart
type TimelinePoint struct {
	Date     string `json:"date"`
	Given    int    `json:"given"`
	Received int    `json:"received"`
}

// Date-bounded queries on beers. They are package-level so store_plan_test.go
// can EXPLAIN exactly the SQL that runs. All take the [lo, hi) bounds from
// dateBounds/yearBounds.
const (
	qCountGiven    = `SELECT COALESCE(SUM(count), 0) FROM beers WHERE giver_id = ? AND ts_rfc >= ? AND ts_rfc < ?`
	qCountReceived = `SELECT COALESCE(SUM(count), 0) FROM beers WHERE recipient_id = ? AND ts_rfc >= ? AND ts_rfc < ?`
	qQuarterly     = `SELECT CAST(substr(ts_rfc, 1, 4) AS INTEGER) AS year,
			(CAST(substr(ts_rfc, 6, 2) AS INTEGER) + 2) / 3 AS quarter,
			SUM(count)
		FROM beers
		WHERE ts_rfc >= ? AND ts_rfc < ?
		GROUP BY year, quarter
		ORDER BY year, quarter`
	qHeatmap = `SELECT substr(ts_rfc, 1, 10) AS date, SUM(count)
		FROM beers
		WHERE ts_rfc >= ? AND ts_rfc < ?
		GROUP BY date
		ORDER BY date`
	qPairs = `SELECT giver_id, recipient_id, SUM(count) AS total
		FROM beers
		WHERE ts_rfc >= ? AND ts_rfc < ?
		GROUP BY giver_id, recipient_id
		ORDER BY total DESC, giver_id, recipient_id
		LIMIT ?`
)

// timelineQuery groups the beers in [lo, hi) by day, week or month.
func timelineQuery(granularity string) string {
	return `SELECT ` + timelinePeriod(granularity) + ` AS period, SUM(count)
		FROM beers WHERE ts_rfc >= ? AND ts_rfc < ?
		GROUP BY period ORDER BY period`
}

// topUsersQuery ranks column (giver_id or recipient_id, never user input) by
// beers in [lo, hi). Ties are ordered by user ID so results are stable.
func topUsersQuery(column string) string {
	return `SELECT ` + column + `, SUM(count) AS total
		FROM beers
		WHERE ts_rfc >= ? AND ts_rfc < ?
		GROUP BY ` + column + `
		ORDER BY total DESC, ` + column + `
		LIMIT ?`
}

// timelinePeriod returns the SQL expression that groups ts_rfc by granularity.
func timelinePeriod(granularity string) string {
	switch granularity {
	case "week":
		// Week number with Monday as first day (strftime %W)
		return `strftime('%Y-W%W', substr(ts_rfc, 1, 10))`
	case "month":
		return `substr(ts_rfc, 1, 7)`
	default: // "day"
		return `substr(ts_rfc, 1, 10)`
	}
}

// GetTimelineStats returns aggregated beer counts grouped by date within a range.
// Granularity can be "day", "week", or "month".
//
// Every beer is both given and received, so Given and Received are the same
// total (one pass over the index).
func (s *SQLiteStore) GetTimelineStats(start, end time.Time, granularity string) ([]TimelinePoint, error) {
	lo, hi := dateBounds(start, end)
	rows, err := s.db.Query(timelineQuery(granularity), lo, hi)
	if err != nil {
		return nil, fmt.Errorf("timeline query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []TimelinePoint
	for rows.Next() {
		var p TimelinePoint
		if err := rows.Scan(&p.Date, &p.Given); err != nil {
			return nil, fmt.Errorf("timeline scan: %w", err)
		}
		p.Received = p.Given
		results = append(results, p)
	}
	return results, rows.Err()
}

// QuarterlyStats represents stats for a single quarter
type QuarterlyStats struct {
	Year    int `json:"year"`
	Quarter int `json:"quarter"`
	Count   int `json:"count"`
}

// GetQuarterlyStats returns beer counts aggregated by quarter for a range of years
func (s *SQLiteStore) GetQuarterlyStats(startYear, endYear int) ([]QuarterlyStats, error) {
	lo, hi := yearBounds(startYear, endYear)
	rows, err := s.db.Query(qQuarterly, lo, hi)
	if err != nil {
		return nil, fmt.Errorf("quarterly query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []QuarterlyStats
	for rows.Next() {
		var q QuarterlyStats
		if err := rows.Scan(&q.Year, &q.Quarter, &q.Count); err != nil {
			return nil, fmt.Errorf("quarterly scan: %w", err)
		}
		results = append(results, q)
	}
	return results, rows.Err()
}

// TopUserStats represents a user with their beer count
type TopUserStats struct {
	UserID string `json:"userId"`
	Count  int    `json:"count"`
}

// TopUsersResult contains top givers and recipients
type TopUsersResult struct {
	Givers     []TopUserStats `json:"givers"`
	Recipients []TopUserStats `json:"recipients"`
}

// GetTopUsers returns the top N givers and recipients in a date range
func (s *SQLiteStore) GetTopUsers(start, end time.Time, limit int) (*TopUsersResult, error) {
	lo, hi := dateBounds(start, end)
	givers, err := s.topUsers("giver_id", lo, hi, limit)
	if err != nil {
		return nil, fmt.Errorf("top givers: %w", err)
	}
	recipients, err := s.topUsers("recipient_id", lo, hi, limit)
	if err != nil {
		return nil, fmt.Errorf("top recipients: %w", err)
	}
	return &TopUsersResult{Givers: givers, Recipients: recipients}, nil
}

// topUsers returns the top limit rows of topUsersQuery(column).
func (s *SQLiteStore) topUsers(column, lo, hi string, limit int) ([]TopUserStats, error) {
	rows, err := s.db.Query(topUsersQuery(column), lo, hi, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []TopUserStats
	for rows.Next() {
		var u TopUserStats
		if err := rows.Scan(&u.UserID, &u.Count); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// HeatmapPoint represents a single day's activity for the calendar heatmap
type HeatmapPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetHeatmapStats returns daily beer counts for a calendar heatmap view
func (s *SQLiteStore) GetHeatmapStats(start, end time.Time) ([]HeatmapPoint, error) {
	lo, hi := dateBounds(start, end)
	rows, err := s.db.Query(qHeatmap, lo, hi)
	if err != nil {
		return nil, fmt.Errorf("heatmap query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []HeatmapPoint
	for rows.Next() {
		var p HeatmapPoint
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			return nil, fmt.Errorf("heatmap scan: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// PairStats represents a giver-recipient pair with their total beer count
type PairStats struct {
	Giver     string `json:"giver"`
	Recipient string `json:"recipient"`
	Count     int    `json:"count"`
}

// GetPairStats returns the top giver→recipient pairs for network visualization
func (s *SQLiteStore) GetPairStats(start, end time.Time, limit int) ([]PairStats, error) {
	lo, hi := dateBounds(start, end)
	rows, err := s.db.Query(qPairs, lo, hi, limit)
	if err != nil {
		return nil, fmt.Errorf("pairs query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []PairStats
	for rows.Next() {
		var p PairStats
		if err := rows.Scan(&p.Giver, &p.Recipient, &p.Count); err != nil {
			return nil, fmt.Errorf("pairs scan: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}
