package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Cache lifetimes for stats responses. Stats of a period that has ended change only when Slack
// delivers a late event, so they may be cached for a while; stats that include today move with
// every beer. The backend sends the headers, the Next.js proxy has to forward them.
const (
	cacheTTLClosedPeriod  = time.Hour
	cacheTTLCurrentPeriod = 30 * time.Second
)

// clock returns the current time; tests inject h.now.
func (h *APIHandlers) clock() time.Time {
	if h.now != nil {
		return h.now()
	}
	return time.Now()
}

// rangeCacheTTL returns how long a stats response for a range ending on end may be cached.
func (h *APIHandlers) rangeCacheTTL(end time.Time) time.Duration {
	now := h.clock().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if endDay.Before(today) {
		return cacheTTLClosedPeriod
	}
	return cacheTTLCurrentPeriod
}

// yearCacheTTL is rangeCacheTTL for series that end with a whole year.
func (h *APIHandlers) yearCacheTTL(endYear int) time.Duration {
	if endYear < h.clock().UTC().Year() {
		return cacheTTLClosedPeriod
	}
	return cacheTTLCurrentPeriod
}

// respondJSON writes v as 200 JSON with ETag and Cache-Control (see writeJSON).
func (h *APIHandlers) respondJSON(w http.ResponseWriter, r *http.Request, handler string, v any, ttl time.Duration) {
	h.writeJSON(w, r, handler, http.StatusOK, v, ttl)
}

// writeJSON encodes v before writing anything, so an encoding error can still become a 500.
// Successful responses (200) with ttl > 0 get a strong ETag and `Cache-Control: private,
// max-age=<ttl>` (private: the endpoints need the bearer token); a matching If-None-Match
// is answered with 304 and no body. Other responses are sent with `Cache-Control: no-store`.
func (h *APIHandlers) writeJSON(w http.ResponseWriter, r *http.Request, handler string, status int, v any, ttl time.Duration) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		h.logger.Error().Str("handler", handler).Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusOK && ttl > 0 {
		sum := sha256.Sum256(buf.Bytes())
		etag := `"` + hex.EncodeToString(sum[:16]) + `"`
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "private, max-age="+strconv.Itoa(int(ttl/time.Second)))
		if etagMatches(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// etagMatches implements the weak comparison of If-None-Match for a single ETag.
func etagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

// namedTask is one unit of work for runParallel.
type namedTask struct {
	name string
	fn   func() error
}

// runParallel runs all tasks concurrently, waits for every one of them and returns the error of
// the first failing task in list order (nil if none failed).
func runParallel(tasks []namedTask) error {
	errs := make([]error, len(tasks))
	var wg sync.WaitGroup
	for i, t := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = t.fn()
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return &taskError{name: tasks[i].name, err: err}
		}
	}
	return nil
}

// taskError names the task that failed.
type taskError struct {
	name string
	err  error
}

func (e *taskError) Error() string { return e.name + ": " + e.err.Error() }
func (e *taskError) Unwrap() error { return e.err }
