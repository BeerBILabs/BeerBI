package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// userCacheQueryChunk keeps the number of bound parameters per query well below
// SQLite's limit (999 in older builds).
const userCacheQueryChunk = 500

// GetCachedUsers returns the user_cache rows for the given IDs, keyed by user ID.
// IDs without a row are absent from the map.
func (s *SQLiteStore) GetCachedUsers(userIDs []string) (map[string]CachedUser, error) {
	out := make(map[string]CachedUser, len(userIDs))
	for start := 0; start < len(userIDs); start += userCacheQueryChunk {
		end := min(start+userCacheQueryChunk, len(userIDs))
		chunk := userIDs[start:end]

		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		query := `SELECT user_id, real_name, profile_image, updated_at FROM user_cache WHERE user_id IN (?` +
			strings.Repeat(",?", len(chunk)-1) + `)`

		if err := s.scanCachedUsers(query, args, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *SQLiteStore) scanCachedUsers(query string, args []any, out map[string]CachedUser) error {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("user cache query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var u CachedUser
		var profileImage, updatedAt sql.NullString
		if err := rows.Scan(&u.UserID, &u.RealName, &profileImage, &updatedAt); err != nil {
			return fmt.Errorf("user cache scan: %w", err)
		}
		u.ProfileImage = profileImage.String
		u.UpdatedAt = parseCacheTime(updatedAt.String)
		out[u.UserID] = u
	}
	return rows.Err()
}

// SetCachedUsers upserts several users into user_cache in one transaction.
func (s *SQLiteStore) SetCachedUsers(users map[string]UserInfo) error {
	if len(users) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("user cache begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT INTO user_cache (user_id, real_name, profile_image, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET real_name = excluded.real_name, profile_image = excluded.profile_image, updated_at = excluded.updated_at`)
	if err != nil {
		return fmt.Errorf("user cache prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	for id, u := range users {
		if _, err := stmt.Exec(id, u.RealName, u.ProfileImage, now); err != nil {
			return fmt.Errorf("user cache upsert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("user cache commit: %w", err)
	}
	return nil
}

// parseCacheTime parses updated_at values. Drivers return DATETIME columns either
// as the stored RFC3339 text or as a formatted time, so accept both. An unparsable
// value yields the zero time, which callers treat as stale.
func parseCacheTime(v string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05Z07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Time{}
}
