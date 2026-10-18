package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// APIHandlers holds dependencies for HTTP handlers
type APIHandlers struct {
	store        *SQLiteStore
	slackClient  *slack.Client
	slackManager *SlackConnectionManager
	redisCache   *RedisUserCache
	users        *UserResolver
	logger       zerolog.Logger
	// now is the clock used for range matching and cache headers; nil means time.Now.
	// Tests inject a fixed clock.
	now func() time.Time
}

// NewAPIHandlers creates a new APIHandlers instance
func NewAPIHandlers(store *SQLiteStore, slackClient *slack.Client, slackManager *SlackConnectionManager, redisCache *RedisUserCache, logger zerolog.Logger) *APIHandlers {
	return &APIHandlers{
		store:        store,
		slackClient:  slackClient,
		slackManager: slackManager,
		redisCache:   redisCache,
		users:        newUserResolverFor(store, redisCache, slackClient, logger),
		logger:       logger,
	}
}

// givenResponse is the JSON body of GET /api/given.
type givenResponse struct {
	User  string `json:"user"`
	Start string `json:"start"`
	End   string `json:"end"`
	Given int    `json:"given"`
}

// receivedResponse is the JSON body of GET /api/received.
type receivedResponse struct {
	User     string `json:"user"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Received int    `json:"received"`
}

// newUserResolverFor wires the resolver from the concrete dependencies. Nil
// pointers become nil interfaces so the resolver skips that layer.
func newUserResolverFor(store *SQLiteStore, redisCache *RedisUserCache, slackClient *slack.Client, logger zerolog.Logger) *UserResolver {
	var (
		st userCacheStore
		rc userRedisCache
		sl SlackUserAPI
	)
	if store != nil {
		st = store
	}
	if redisCache != nil {
		rc = redisCache
	}
	if slackClient != nil {
		sl = slackClient
	}
	return NewUserResolver(st, rc, sl, logger)
}

// GivenHandler returns the number of beers given by a user in a date range
func (h *APIHandlers) GivenHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "given").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	user := r.URL.Query().Get("user")
	if user == "" {
		h.logger.Warn().Str("handler", "given").Msg("missing user parameter")
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}
	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "given").Str("user", user).Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}
	c, err := h.store.CountGivenInDateRange(user, start, end)
	if err != nil {
		h.logger.Error().Str("handler", "given").Str("user", user).Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "given").Str("user", user).Int("count", c).Msg("request completed")
	h.respondJSON(w, r, "given", givenResponse{
		User:  user,
		Start: start.Format(dateLayout),
		End:   end.Format(dateLayout),
		Given: c,
	}, h.rangeCacheTTL(end))
}

// ReceivedHandler returns the number of beers received by a user in a date range
func (h *APIHandlers) ReceivedHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "received").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	user := r.URL.Query().Get("user")
	if user == "" {
		h.logger.Warn().Str("handler", "received").Msg("missing user parameter")
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}
	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "received").Str("user", user).Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}
	c, err := h.store.CountReceivedInDateRange(user, start, end)
	if err != nil {
		h.logger.Error().Str("handler", "received").Str("user", user).Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "received").Str("user", user).Int("count", c).Msg("request completed")
	h.respondJSON(w, r, "received", receivedResponse{
		User:     user,
		Start:    start.Format(dateLayout),
		End:      end.Format(dateLayout),
		Received: c,
	}, h.rangeCacheTTL(end))
}

// UserHandler returns the display data of one user (Redis, SQLite cache, then Slack)
func (h *APIHandlers) UserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "user").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	userID := r.URL.Query().Get("user")
	if userID == "" {
		h.logger.Warn().Str("handler", "user").Msg("missing user parameter")
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}

	// Redis -> SQLite user_cache -> Slack users.info (see users.go).
	info, ok := h.users.ResolveOne(r.Context(), userID, usersRequestWait)
	if !ok {
		h.logger.Warn().Str("handler", "user").Str("userID", userID).Msg("user not found in cache or Slack")
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	h.logger.Debug().Str("handler", "user").Str("userID", userID).Msg("request completed")
	h.writeJSON(w, r, "user", http.StatusOK, info, 0)
}

// HealthHandler returns the health status of the service
func (h *APIHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "health").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	health := map[string]interface{}{
		"status":          "healthy",
		"service":         "beerbot-backend",
		"slack_connected": h.slackManager.IsConnected(),
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	// Test Slack connection if requested
	if r.URL.Query().Get("check_slack") == "true" {
		if err := h.slackManager.TestConnection(r.Context()); err != nil {
			h.logger.Warn().Str("handler", "health").Err(err).Msg("slack connection test failed")
			health["slack_connection_error"] = err.Error()
			health["status"] = "degraded"
		}
	}

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	h.logger.Info().Str("handler", "health").Str("status", health["status"].(string)).Msg("request completed")
	h.writeJSON(w, r, "health", statusCode, health, 0)
}

// ============================================================================
// Stats Handlers for Analytics/BI Features
// ============================================================================

// TimelineHandler returns aggregated beer counts over time
// Query params: start, end (YYYY-MM-DD), granularity (day|week|month)
func (h *APIHandlers) TimelineHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "timeline").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "timeline").Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "day"
	}
	if granularity != "day" && granularity != "week" && granularity != "month" {
		http.Error(w, "granularity must be day, week, or month", http.StatusBadRequest)
		return
	}

	data, err := h.store.GetTimelineStats(start, end, granularity)
	if err != nil {
		h.logger.Error().Str("handler", "timeline").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "timeline").Int("points", len(data)).Str("granularity", granularity).Msg("request completed")
	h.respondJSON(w, r, "timeline", data, h.rangeCacheTTL(end))
}

// QuarterlyHandler returns beer counts aggregated by quarter
// Query params: start_year, end_year (integers)
func (h *APIHandlers) QuarterlyHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "quarterly").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	startYear, endYear := h.parseYearRange(r)

	data, err := h.store.GetQuarterlyStats(startYear, endYear)
	if err != nil {
		h.logger.Error().Str("handler", "quarterly").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "quarterly").Int("quarters", len(data)).Msg("request completed")
	h.respondJSON(w, r, "quarterly", data, h.yearCacheTTL(endYear))
}

// TopUsersHandler returns top N givers and recipients
// Query params: start, end (YYYY-MM-DD), limit (default 20)
func (h *APIHandlers) TopUsersHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "top").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "top").Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	data, err := h.topUsers(r.Context(), "top", start, end, limit)
	if err != nil {
		h.logger.Error().Str("handler", "top").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "top").Int("givers", len(data.Givers)).Int("recipients", len(data.Recipients)).Msg("request completed")
	h.respondJSON(w, r, "top", h.withUserNamesTop(r.Context(), data), h.rangeCacheTTL(end))
}

// HeatmapHandler returns daily beer counts for calendar heatmap
// Query params: start, end (YYYY-MM-DD)
func (h *APIHandlers) HeatmapHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "heatmap").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "heatmap").Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	data, err := h.store.GetHeatmapStats(start, end)
	if err != nil {
		h.logger.Error().Str("handler", "heatmap").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "heatmap").Int("days", len(data)).Msg("request completed")
	h.respondJSON(w, r, "heatmap", data, h.rangeCacheTTL(end))
}

// PairsHandler returns giver→recipient pairs for network visualization
// Query params: start, end (YYYY-MM-DD), limit (default 30)
func (h *APIHandlers) PairsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "pairs").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "pairs").Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	data, err := h.store.GetPairStats(start, end, limit)
	if err != nil {
		h.logger.Error().Str("handler", "pairs").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "pairs").Int("pairs", len(data)).Msg("request completed")
	h.respondJSON(w, r, "pairs", h.withUserNamesPairs(r.Context(), data), h.rangeCacheTTL(end))
}

// BatchUsersHandler returns information for multiple users in one request
// Query params: ids (comma-separated user IDs, max 100)
func (h *APIHandlers) BatchUsersHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "batch_users").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		h.logger.Warn().Str("handler", "batch_users").Msg("missing ids parameter")
		http.Error(w, "ids required (comma-separated)", http.StatusBadRequest)
		return
	}

	// Parse and validate IDs
	userIDs := []string{}
	for _, id := range strings.Split(idsParam, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			userIDs = append(userIDs, id)
		}
	}

	if len(userIDs) == 0 {
		h.logger.Warn().Str("handler", "batch_users").Msg("no valid user IDs provided")
		http.Error(w, "no valid user IDs provided", http.StatusBadRequest)
		return
	}

	// Cap at batch size limit (max 100 users per request)
	const batchSizeLimit = 100
	if len(userIDs) > batchSizeLimit {
		h.logger.Warn().
			Str("handler", "batch_users").
			Int("count", len(userIDs)).
			Int("limit", batchSizeLimit).
			Msg("too many user IDs, capping at batch size limit")
		userIDs = userIDs[:batchSizeLimit]
	}

	// Redis -> SQLite user_cache -> Slack users.info on a bounded worker pool.
	// Lookups still running after usersRequestWait finish in the background.
	results := h.users.Resolve(r.Context(), userIDs, usersRequestWait)

	h.logger.Info().Str("handler", "batch_users").Int("requested", len(userIDs)).Int("found", len(results)).Msg("request completed")
	// No caching: the result may be partial while lookups are still running.
	h.writeJSON(w, r, "batch_users", http.StatusOK, results, 0)
}

// matchDateRangeToCache attempts to match a date range to a cached range key
// Returns empty string if no match (caller should fall back to database)
func (h *APIHandlers) matchDateRangeToCache(start, end time.Time) string {
	now := h.clock()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Today
	if isSameDay(start, today) && isSameDay(end, today) {
		return RangeToday
	}

	// Last 7 days
	sevenDaysAgo := today.AddDate(0, 0, -6)
	if isSameDay(start, sevenDaysAgo) && isSameDay(end, today) {
		return RangeLast7Days
	}

	// Current month
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if isSameDay(start, monthStart) && isSameDay(end, today) {
		return RangeCurrentMonth
	}

	// Last month
	lastMonthStart := monthStart.AddDate(0, -1, 0)
	lastMonthEnd := monthStart.AddDate(0, 0, -1)
	if isSameDay(start, lastMonthStart) && isSameDay(end, lastMonthEnd) {
		return RangeLastMonth
	}

	// Current quarter
	quarter := getQuarterNumber(now)
	quarterStart := time.Date(now.Year(), getQuarterStartMonth(quarter), 1, 0, 0, 0, 0, now.Location())
	if isSameDay(start, quarterStart) && isSameDay(end, today) {
		return RangeCurrentQuarter
	}

	// Last quarter
	lastQuarterNum := quarter - 1
	lastQuarterYear := now.Year()
	if lastQuarterNum == 0 {
		lastQuarterNum = 4
		lastQuarterYear--
	}
	lastQuarterStart := time.Date(lastQuarterYear, getQuarterStartMonth(lastQuarterNum), 1, 0, 0, 0, 0, now.Location())
	lastQuarterEnd := quarterStart.AddDate(0, 0, -1)
	if isSameDay(start, lastQuarterStart) && isSameDay(end, lastQuarterEnd) {
		return RangeLastQuarter
	}

	// All-time (anything from 2020 or earlier to today)
	if start.Year() <= 2020 && isSameDay(end, today) {
		return RangeAllTime
	}

	return "" // No match - use database
}

// isSameDay checks if two times are on the same calendar day
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// CombinedAnalyticsResponse holds all analytics data for a date range
type CombinedAnalyticsResponse struct {
	Timeline      []TimelinePoint  `json:"timeline"`
	TopGivers     []TopUserStats   `json:"top_givers"`
	TopRecipients []TopUserStats   `json:"top_recipients"`
	Heatmap       []HeatmapPoint   `json:"heatmap"`
	Pairs         []PairStats      `json:"pairs"`
	Quarterly     []QuarterlyStats `json:"quarterly"`
}

// CombinedAnalyticsHandler returns all analytics data in one request. The independent
// queries run concurrently (they only read, and SQLite serves concurrent readers).
// Query params: start, end (YYYY-MM-DD), granularity (day|week|month), limit (default 20),
// pairs_limit (default 15), start_year, end_year (quarterly series, same defaults as
// /api/stats/quarterly)
func (h *APIHandlers) CombinedAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "combined_analytics").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	start, end, err := parseDateRangeFromParams(r)
	if err != nil {
		h.logger.Warn().Str("handler", "combined_analytics").Err(err).Msg("invalid date range")
		http.Error(w, "invalid or missing date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Parse parameters
	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "day"
	}
	if granularity != "day" && granularity != "week" && granularity != "month" {
		http.Error(w, "granularity must be day, week, or month", http.StatusBadRequest)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	pairsLimit := 15
	if pairsLimitStr := r.URL.Query().Get("pairs_limit"); pairsLimitStr != "" {
		if v, err := strconv.Atoi(pairsLimitStr); err == nil && v > 0 && v <= 50 {
			pairsLimit = v
		}
	}

	startYear, endYear := h.parseYearRange(r)

	// Every task writes only its own field of response, so no locking is needed.
	response := CombinedAnalyticsResponse{}
	err = runParallel([]namedTask{
		{"timeline", func() (err error) {
			response.Timeline, err = h.store.GetTimelineStats(start, end, granularity)
			return err
		}},
		{"top users", func() error {
			topUsers, err := h.topUsers(r.Context(), "combined_analytics", start, end, limit)
			if err != nil {
				return err
			}
			response.TopGivers = topUsers.Givers
			response.TopRecipients = topUsers.Recipients
			return nil
		}},
		{"heatmap", func() (err error) {
			response.Heatmap, err = h.store.GetHeatmapStats(start, end)
			return err
		}},
		{"pairs", func() (err error) {
			response.Pairs, err = h.store.GetPairStats(start, end, pairsLimit)
			return err
		}},
		{"quarterly", func() (err error) {
			response.Quarterly, err = h.store.GetQuarterlyStats(startYear, endYear)
			return err
		}},
	})
	if err != nil {
		h.logger.Error().Str("handler", "combined_analytics").Err(err).Msg("failed to fetch analytics")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().
		Str("handler", "combined_analytics").
		Int("timeline_points", len(response.Timeline)).
		Int("top_givers", len(response.TopGivers)).
		Int("top_recipients", len(response.TopRecipients)).
		Int("heatmap_days", len(response.Heatmap)).
		Int("pairs", len(response.Pairs)).
		Int("quarters", len(response.Quarterly)).
		Msg("request completed")

	// The response is as fresh as its newest series: the quarterly series covers whole years
	// (by default up to the current one), so a closed date range alone does not make it final.
	h.respondJSON(w, r, "combined_analytics", h.withUserNamesCombined(r.Context(), response), min(h.rangeCacheTTL(end), h.yearCacheTTL(endYear)))
}

// topUsers returns the top givers and recipients for a range: from Redis for the common
// cached ranges, otherwise (or on a cache miss or error) from SQLite.
func (h *APIHandlers) topUsers(ctx context.Context, handler string, start, end time.Time, limit int) (*TopUsersResult, error) {
	rangeKey := h.matchDateRangeToCache(start, end)
	if rangeKey != "" && h.redisCache != nil {
		givers, errG := h.redisCache.GetTopGivers(ctx, rangeKey, limit)
		recipients, errR := h.redisCache.GetTopRecipients(ctx, rangeKey, limit)

		if errG == nil && errR == nil && givers != nil && recipients != nil {
			h.logger.Debug().Str("handler", handler).Str("range", rangeKey).Msg("redis cache hit")
			return &TopUsersResult{Givers: givers, Recipients: recipients}, nil
		}
	}
	h.logger.Debug().Str("handler", handler).Msg("reading top users from database")
	return h.store.GetTopUsers(start, end, limit)
}

// parseYearRange reads start_year and end_year (default: the current year and the two before).
// Unparsable values fall back to the default.
func (h *APIHandlers) parseYearRange(r *http.Request) (startYear, endYear int) {
	currentYear := h.clock().Year()
	startYear = currentYear - 2
	endYear = currentYear
	if v, err := strconv.Atoi(r.URL.Query().Get("start_year")); err == nil {
		startYear = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("end_year")); err == nil {
		endYear = v
	}
	return startYear, endYear
}
