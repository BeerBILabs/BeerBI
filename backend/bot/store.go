package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

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

func (s *SQLiteStore) migrate() error {
	// Ensure simple auxiliary tables exist
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
	// ...existing code...
	// After beers table creation and schema validation, create indexes
	indexStmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_beers_giver_id_ts_rfc ON beers (giver_id, ts_rfc);`,
		`CREATE INDEX IF NOT EXISTS idx_beers_recipient_id_ts_rfc ON beers (recipient_id, ts_rfc);`,
		`CREATE INDEX IF NOT EXISTS idx_beers_emoji_ts_rfc ON beers (ts_rfc);`,
		`CREATE INDEX IF NOT EXISTS idx_emoji_counts_user_id_emoji ON emoji_counts (user_id, emoji);`,
	}
	// Only run index creation if beers table exists now
	var beersExists int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='beers'`).Scan(&beersExists); err == nil && beersExists > 0 {
		for _, st := range indexStmts {
			if _, err := s.db.Exec(st); err != nil {
				return fmt.Errorf("migrate index exec: %w", err)
			}
		}
	}

	// Desired beers table create statement
	desiredCreate := `CREATE TABLE beers (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            giver_id TEXT NOT NULL,
            recipient_id TEXT NOT NULL,
            ts TEXT NOT NULL, -- original Slack ts string (with fraction)
            ts_rfc DATETIME NOT NULL, -- parsed RFC3339 time for date queries
            count INTEGER NOT NULL DEFAULT 1,
            UNIQUE (giver_id, recipient_id, ts)
        );`

	// If beers table doesn't exist, create it with the desired schema
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='beers'`).Scan(&exists); err != nil {
		return fmt.Errorf("migrate check beers exists: %w", err)
	}
	if exists == 0 {
		if _, err := s.db.Exec(desiredCreate); err != nil {
			return fmt.Errorf("migrate create beers: %w", err)
		}
		return nil
	}

	// beers table exists: ensure required columns and constraints
	// collect existing columns
	cols := map[string]bool{}
	rows, err := s.db.Query(`PRAGMA table_info(beers);`)
	if err != nil {
		return fmt.Errorf("migrate pragma: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("migrate scan pragma: %w", err)
		}
		cols[name] = true
	}

	// Add missing columns non-destructively
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

	// Ensure UNIQUE(giver_id, recipient_id, ts) exists. SQLite doesn't support adding
	// UNIQUE constraints via ALTER, so if it's missing we recreate the table non-destructively
	// by aggregating existing rows into the desired schema.
	var createSQL sql.NullString
	if err := s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='beers'`).Scan(&createSQL); err != nil {
		return fmt.Errorf("migrate select create sql: %w", err)
	}
	if !createSQL.Valid || !strings.Contains(strings.ToUpper(createSQL.String), "UNIQUE") {
		// Recreate table: create beers_new, copy aggregated data, swap tables
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("migrate begin tx: %w", err)
		}
		// create new table with desired schema
		if _, err := tx.Exec(desiredCreate); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate create beers_new: %w", err)
		}
		// copy aggregated data into beers (treat missing count as 1 and compute ts_rfc if NULL)
		copyStmt := `INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count)
            SELECT giver_id, recipient_id, ts,
                COALESCE(ts_rfc, datetime(substr(ts,1,instr(ts,'.')-1), 'unixepoch')),
                COALESCE(SUM(count), COUNT(1))
            FROM (SELECT * FROM beers) GROUP BY giver_id, recipient_id, ts;`
		if _, err := tx.Exec(copyStmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate copy aggregated: %w", err)
		}
		// drop old table and keep the new one under the original name
		if _, err := tx.Exec(`DROP TABLE IF EXISTS beers;`); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate drop old beers: %w", err)
		}
		if _, err := tx.Exec(`ALTER TABLE beers RENAME TO beers_old;`); err == nil {
			// if rename succeeded unexpectedly, try to rename back
		}
		// Note: desiredCreate created a table named 'beers' already; we dropped old table, so commit.
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate commit recreate: %w", err)
		}
	}

	return nil
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

func (s *SQLiteStore) IncEmoji(userID, emoji string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

// AddBeer inserts a beer event (one record per beer)
// AddBeer records a beer-gift event for a single message: it inserts or upserts
// a row with the provided count. If the same (giver, recipient, ts) already
// exists, the count will be updated to the provided value (last write wins).
// AddBeer records a beer-gift event for a single message: it inserts or upserts
// a row with the provided count keyed by the original Slack ts string (ts).
func (s *SQLiteStore) AddBeer(giverID, recipientID string, slackTs string, t time.Time, count int) error {
	_, err := s.db.Exec(`INSERT INTO beers (giver_id, recipient_id, ts, ts_rfc, count) VALUES (?, ?, ?, ?, ?) ON CONFLICT(giver_id, recipient_id, ts) DO UPDATE SET count = excluded.count`, giverID, recipientID, slackTs, t.UTC().Format(time.RFC3339), count)
	return err
}

// CountGivenInDateRange returns how many beers the giver gave in the given date range
func (s *SQLiteStore) CountGivenInDateRange(giverID string, start time.Time, end time.Time) (int, error) {
	// Use YYYY-MM-DD format for SQLite date() comparison
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	var c int
	query := `SELECT COALESCE(SUM(count), 0) FROM beers WHERE giver_id = ? AND substr(ts_rfc, 1, 10) BETWEEN ? AND ?`
	err := s.db.QueryRow(query, giverID, startStr, endStr).Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}

// CountReceivedInDateRange returns total beers received by recipient in the given date range
func (s *SQLiteStore) CountReceivedInDateRange(recipientID string, start time.Time, end time.Time) (int, error) {
	var c int
	// Use YYYY-MM-DD format for SQLite date() comparison
	query := `SELECT COALESCE(SUM(count), 0) FROM beers WHERE recipient_id = ? AND substr(ts_rfc, 1, 10) BETWEEN ? AND ?`
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	err := s.db.QueryRow(query, recipientID, startStr, endStr).Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}

// CountGivenOnDate returns how many beers the giver gave on the given date (YYYY-MM-DD)
func (s *SQLiteStore) CountGivenOnDate(giverID string, date string) (int, error) {
	t, err := time.Parse("2006-01-02", date)
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
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}
	return s.CountReceivedInDateRange(recipientID, t, t)
}

// GetAllGivers returns the list of all distinct user IDs that have given at least one beer.
func (s *SQLiteStore) GetAllGivers() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT giver_id FROM beers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// GetAllRecipients returns the list of all distinct recipient user IDs that have received at least one beer.
func (s *SQLiteStore) GetAllRecipients() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT recipient_id FROM beers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
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
	var updatedAt string
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
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
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
// ============================================================================

// TimelinePoint represents a single data point in a timeline chart
type TimelinePoint struct {
	Date     string `json:"date"`
	Given    int    `json:"given"`
	Received int    `json:"received"`
}

// GetTimelineStats returns aggregated beer counts grouped by date within a range.
// Granularity can be "day", "week", or "month".
func (s *SQLiteStore) GetTimelineStats(start, end time.Time, granularity string) ([]TimelinePoint, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	fmt.Printf("[STORE] GetTimelineStats: start=%s end=%s granularity=%s\n", startStr, endStr, granularity)

	var dateExpr string
	switch granularity {
	case "week":
		// Group by ISO week (Monday start) - use strftime %W for week number
		dateExpr = `strftime('%Y-W%W', substr(ts_rfc, 1, 10))`
	case "month":
		dateExpr = `strftime('%Y-%m', substr(ts_rfc, 1, 10))`
	default: // "day"
		dateExpr = `substr(ts_rfc, 1, 10)`
	}

	query := fmt.Sprintf(`
		WITH dates AS (
			SELECT DISTINCT %s as period FROM beers 
			WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
		),
		given_counts AS (
			SELECT %s as period, COALESCE(SUM(count), 0) as total
			FROM beers WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
			GROUP BY %s
		),
		received_counts AS (
			SELECT %s as period, COALESCE(SUM(count), 0) as total
			FROM beers WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
			GROUP BY %s
		)
		SELECT d.period, COALESCE(g.total, 0), COALESCE(r.total, 0)
		FROM dates d
		LEFT JOIN given_counts g ON d.period = g.period
		LEFT JOIN received_counts r ON d.period = r.period
		ORDER BY d.period
	`, dateExpr, dateExpr, dateExpr, dateExpr, dateExpr)

	fmt.Printf("[STORE] GetTimelineStats query: %s\n", query)
	rows, err := s.db.Query(query, startStr, endStr, startStr, endStr, startStr, endStr)
	if err != nil {
		fmt.Printf("[STORE] GetTimelineStats query error: %v\n", err)
		return nil, fmt.Errorf("timeline query: %w", err)
	}
	defer rows.Close()

	var results []TimelinePoint
	for rows.Next() {
		var p TimelinePoint
		if err := rows.Scan(&p.Date, &p.Given, &p.Received); err != nil {
			fmt.Printf("[STORE] GetTimelineStats scan error: %v\n", err)
			return nil, fmt.Errorf("timeline scan: %w", err)
		}
		results = append(results, p)
	}
	fmt.Printf("[STORE] GetTimelineStats returning %d results\n", len(results))
	return results, nil
}

// QuarterlyStats represents stats for a single quarter
type QuarterlyStats struct {
	Year    int `json:"year"`
	Quarter int `json:"quarter"`
	Count   int `json:"count"`
}

// GetQuarterlyStats returns beer counts aggregated by quarter for a range of years
func (s *SQLiteStore) GetQuarterlyStats(startYear, endYear int) ([]QuarterlyStats, error) {
	fmt.Printf("[STORE] GetQuarterlyStats: startYear=%d endYear=%d\n", startYear, endYear)
	query := `
		SELECT 
			CAST(strftime('%Y', substr(ts_rfc, 1, 10)) AS INTEGER) as year,
			CASE 
				WHEN CAST(strftime('%m', substr(ts_rfc, 1, 10)) AS INTEGER) BETWEEN 1 AND 3 THEN 1
				WHEN CAST(strftime('%m', substr(ts_rfc, 1, 10)) AS INTEGER) BETWEEN 4 AND 6 THEN 2
				WHEN CAST(strftime('%m', substr(ts_rfc, 1, 10)) AS INTEGER) BETWEEN 7 AND 9 THEN 3
				ELSE 4
			END as quarter,
			COALESCE(SUM(count), 0) as total
		FROM beers
		WHERE CAST(strftime('%Y', substr(ts_rfc, 1, 10)) AS INTEGER) BETWEEN ? AND ?
		GROUP BY year, quarter
		ORDER BY year, quarter
	`

	rows, err := s.db.Query(query, startYear, endYear)
	if err != nil {
		fmt.Printf("[STORE] GetQuarterlyStats query error: %v\n", err)
		return nil, fmt.Errorf("quarterly query: %w", err)
	}
	defer rows.Close()

	var results []QuarterlyStats
	for rows.Next() {
		var q QuarterlyStats
		if err := rows.Scan(&q.Year, &q.Quarter, &q.Count); err != nil {
			fmt.Printf("[STORE] GetQuarterlyStats scan error: %v\n", err)
			return nil, fmt.Errorf("quarterly scan: %w", err)
		}
		results = append(results, q)
	}
	fmt.Printf("[STORE] GetQuarterlyStats returning %d results\n", len(results))
	return results, nil
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
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	fmt.Printf("[STORE] GetTopUsers: start=%s end=%s limit=%d\n", startStr, endStr, limit)

	// Get top givers
	giversQuery := `
		SELECT giver_id, COALESCE(SUM(count), 0) as total
		FROM beers
		WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
		GROUP BY giver_id
		ORDER BY total DESC
		LIMIT ?
	`
	giversRows, err := s.db.Query(giversQuery, startStr, endStr, limit)
	if err != nil {
		fmt.Printf("[STORE] GetTopUsers givers query error: %v\n", err)
		return nil, fmt.Errorf("top givers query: %w", err)
	}
	defer giversRows.Close()

	var givers []TopUserStats
	for giversRows.Next() {
		var u TopUserStats
		if err := giversRows.Scan(&u.UserID, &u.Count); err != nil {
			return nil, fmt.Errorf("top givers scan: %w", err)
		}
		givers = append(givers, u)
	}

	// Get top recipients
	recipientsQuery := `
		SELECT recipient_id, COALESCE(SUM(count), 0) as total
		FROM beers
		WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
		GROUP BY recipient_id
		ORDER BY total DESC
		LIMIT ?
	`
	recipientsRows, err := s.db.Query(recipientsQuery, startStr, endStr, limit)
	if err != nil {
		fmt.Printf("[STORE] GetTopUsers recipients query error: %v\n", err)
		return nil, fmt.Errorf("top recipients query: %w", err)
	}
	defer recipientsRows.Close()

	var recipients []TopUserStats
	for recipientsRows.Next() {
		var u TopUserStats
		if err := recipientsRows.Scan(&u.UserID, &u.Count); err != nil {
			return nil, fmt.Errorf("top recipients scan: %w", err)
		}
		recipients = append(recipients, u)
	}

	fmt.Printf("[STORE] GetTopUsers returning %d givers, %d recipients\n", len(givers), len(recipients))
	return &TopUsersResult{Givers: givers, Recipients: recipients}, nil
}

// HeatmapPoint represents a single day's activity for the calendar heatmap
type HeatmapPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetHeatmapStats returns daily beer counts for a calendar heatmap view
func (s *SQLiteStore) GetHeatmapStats(start, end time.Time) ([]HeatmapPoint, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	fmt.Printf("[STORE] GetHeatmapStats: start=%s end=%s\n", startStr, endStr)

	query := `
		SELECT substr(ts_rfc, 1, 10) as date, COALESCE(SUM(count), 0) as total
		FROM beers
		WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
		GROUP BY date
		ORDER BY date
	`

	rows, err := s.db.Query(query, startStr, endStr)
	if err != nil {
		fmt.Printf("[STORE] GetHeatmapStats query error: %v\n", err)
		return nil, fmt.Errorf("heatmap query: %w", err)
	}
	defer rows.Close()

	var results []HeatmapPoint
	for rows.Next() {
		var p HeatmapPoint
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			fmt.Printf("[STORE] GetHeatmapStats scan error: %v\n", err)
			return nil, fmt.Errorf("heatmap scan: %w", err)
		}
		results = append(results, p)
	}
	fmt.Printf("[STORE] GetHeatmapStats returning %d results\n", len(results))
	return results, nil
}

// PairStats represents a giver-recipient pair with their total beer count
type PairStats struct {
	Giver     string `json:"giver"`
	Recipient string `json:"recipient"`
	Count     int    `json:"count"`
}

// GetPairStats returns the top giver→recipient pairs for network visualization
func (s *SQLiteStore) GetPairStats(start, end time.Time, limit int) ([]PairStats, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	fmt.Printf("[STORE] GetPairStats: start=%s end=%s limit=%d\n", startStr, endStr, limit)

	query := `
		SELECT giver_id, recipient_id, COALESCE(SUM(count), 0) as total
		FROM beers
		WHERE substr(ts_rfc, 1, 10) BETWEEN ? AND ?
		GROUP BY giver_id, recipient_id
		ORDER BY total DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, startStr, endStr, limit)
	if err != nil {
		fmt.Printf("[STORE] GetPairStats query error: %v\n", err)
		return nil, fmt.Errorf("pairs query: %w", err)
	}
	defer rows.Close()

	var results []PairStats
	for rows.Next() {
		var p PairStats
		if err := rows.Scan(&p.Giver, &p.Recipient, &p.Count); err != nil {
			fmt.Printf("[STORE] GetPairStats scan error: %v\n", err)
			return nil, fmt.Errorf("pairs scan: %w", err)
		}
		results = append(results, p)
	}
	fmt.Printf("[STORE] GetPairStats returning %d results\n", len(results))
	return results, nil
}
