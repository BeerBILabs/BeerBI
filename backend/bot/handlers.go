package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	logger       zerolog.Logger
}

// NewAPIHandlers creates a new APIHandlers instance
func NewAPIHandlers(store *SQLiteStore, slackClient *slack.Client, slackManager *SlackConnectionManager, redisCache *RedisUserCache, logger zerolog.Logger) *APIHandlers {
	return &APIHandlers{
		store:        store,
		slackClient:  slackClient,
		slackManager: slackManager,
		redisCache:   redisCache,
		logger:       logger,
	}
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
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"user":"%s","start":"%s","end":"%s","given":%d}`,
		user, start.Format("2006-01-02"), end.Format("2006-01-02"), c)))
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
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"user":"%s","start":"%s","end":"%s","received":%d}`,
		user, start.Format("2006-01-02"), end.Format("2006-01-02"), c)))
}

// UserHandler returns user information from Slack (with database caching)
func (h *APIHandlers) UserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "user").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	userID := r.URL.Query().Get("user")
	if userID == "" {
		h.logger.Warn().Str("handler", "user").Msg("missing user parameter")
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}

	var realName, profileImage string
	ctx := r.Context()

	// Try Redis cache first (if available)
	if h.redisCache != nil {
		cached, err := h.redisCache.GetUser(ctx, userID)
		if err == nil && cached != nil {
			h.logger.Info().Str("handler", "user").Str("userID", userID).Msg("redis cache hit")
			realName = cached.RealName
			profileImage = cached.ProfileImage

			response := map[string]string{
				"real_name":     realName,
				"profile_image": profileImage,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Try Slack API to get fresh data
	user, err := h.slackClient.GetUserInfo(userID)
	if err != nil {
		h.logger.Warn().Str("handler", "user").Str("userID", userID).Err(err).Msg("slack API error, checking cache")
	}

	// Check if we got valid data from Slack (non-empty real_name)
	// Deactivated users may return empty real_name even on successful API call
	if err == nil && user.RealName != "" {
		realName = user.RealName
		profileImage = user.Profile.Image192

		// Cache to Redis (if available)
		if h.redisCache != nil {
			if cacheErr := h.redisCache.SetUser(ctx, userID, realName, profileImage); cacheErr != nil {
				h.logger.Warn().Str("handler", "user").Str("userID", userID).Err(cacheErr).Msg("failed to cache user to redis")
			}
		}

		// Cache to SQLite
		if cacheErr := h.store.SetCachedUser(userID, realName, profileImage); cacheErr != nil {
			h.logger.Warn().Str("handler", "user").Str("userID", userID).Err(cacheErr).Msg("failed to cache user to sqlite")
		}
	} else {
		// Slack API failed or returned empty name - try SQLite cache
		cached, cacheErr := h.store.GetCachedUser(userID)
		if cacheErr != nil {
			h.logger.Error().Str("handler", "user").Str("userID", userID).Err(cacheErr).Msg("cache lookup error")
		}
		if cached != nil && cached.RealName != "" {
			h.logger.Info().Str("handler", "user").Str("userID", userID).Str("real_name", cached.RealName).Msg("returning cached user")
			realName = cached.RealName
			profileImage = cached.ProfileImage
		} else if err == nil && user != nil {
			// API succeeded but no cache - return what Slack gave us (even if empty)
			// Also use profile image from Slack if available
			realName = user.RealName
			profileImage = user.Profile.Image192
			h.logger.Warn().Str("handler", "user").Str("userID", userID).Msg("slack returned empty name, no cache available")
		} else {
			// No cache available and API failed, return error
			h.logger.Error().Str("handler", "user").Str("userID", userID).Msg("user not found in Slack or cache")
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
	}

	h.logger.Info().Str("handler", "user").Str("userID", userID).Str("real_name", realName).Msg("request completed")
	response := map[string]string{
		"real_name":     realName,
		"profile_image": profileImage,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(response); err != nil {
		h.logger.Error().Str("handler", "user").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}

// GiversHandler returns the list of all givers
func (h *APIHandlers) GiversHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "givers").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	list, err := h.store.GetAllGivers()
	if err != nil {
		h.logger.Error().Str("handler", "givers").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "givers").Int("count", len(list)).Msg("request completed")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(list); err != nil {
		h.logger.Error().Str("handler", "givers").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}

// RecipientsHandler returns the list of all recipients
func (h *APIHandlers) RecipientsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "recipients").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	list, err := h.store.GetAllRecipients()
	if err != nil {
		h.logger.Error().Str("handler", "recipients").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "recipients").Int("count", len(list)).Msg("request completed")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(list); err != nil {
		h.logger.Error().Str("handler", "recipients").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}

// HealthHandler returns the health status of the service
func (h *APIHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Str("handler", "health").Str("method", r.Method).Str("path", r.URL.Path).Msg("request received")

	w.Header().Set("Content-Type", "application/json")

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
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(health); err != nil {
		h.logger.Error().Str("handler", "health").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
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
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		h.logger.Error().Str("handler", "timeline").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
}

// QuarterlyHandler returns beer counts aggregated by quarter
// Query params: start_year, end_year (integers)
func (h *APIHandlers) QuarterlyHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Str("handler", "quarterly").Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).Msg("request received")

	startYearStr := r.URL.Query().Get("start_year")
	endYearStr := r.URL.Query().Get("end_year")

	currentYear := time.Now().Year()
	startYear := currentYear - 2
	endYear := currentYear

	if startYearStr != "" {
		if v, err := strconv.Atoi(startYearStr); err == nil {
			startYear = v
		}
	}
	if endYearStr != "" {
		if v, err := strconv.Atoi(endYearStr); err == nil {
			endYear = v
		}
	}

	data, err := h.store.GetQuarterlyStats(startYear, endYear)
	if err != nil {
		h.logger.Error().Str("handler", "quarterly").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "quarterly").Int("quarters", len(data)).Msg("request completed")
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		h.logger.Error().Str("handler", "quarterly").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
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

	data, err := h.store.GetTopUsers(start, end, limit)
	if err != nil {
		h.logger.Error().Str("handler", "top").Err(err).Msg("database error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("handler", "top").Int("givers", len(data.Givers)).Int("recipients", len(data.Recipients)).Msg("request completed")
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		h.logger.Error().Str("handler", "top").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
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
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		h.logger.Error().Str("handler", "heatmap").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
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
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		h.logger.Error().Str("handler", "pairs").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
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

	// Cap at 100 IDs
	if len(userIDs) > 100 {
		h.logger.Warn().Str("handler", "batch_users").Int("count", len(userIDs)).Msg("too many user IDs, capping at 100")
		userIDs = userIDs[:100]
	}

	ctx := r.Context()
	results := make(map[string]map[string]string)
	missing := []string{}

	// Try Redis cache first (if available)
	if h.redisCache != nil {
		cached, err := h.redisCache.GetUsers(ctx, userIDs)
		if err == nil {
			for userID, data := range cached {
				results[userID] = map[string]string{
					"real_name":     data.RealName,
					"profile_image": data.ProfileImage,
				}
			}
		}
	}

	// Find missing users
	for _, userID := range userIDs {
		if _, ok := results[userID]; !ok {
			missing = append(missing, userID)
		}
	}

	// Fetch missing users from Slack API
	if len(missing) > 0 {
		h.logger.Debug().Str("handler", "batch_users").Int("missing", len(missing)).Msg("fetching missing users from Slack")

		toCache := make(map[string]UserCacheData)

		for _, userID := range missing {
			user, err := h.slackClient.GetUserInfo(userID)
			if err != nil {
				h.logger.Warn().Str("handler", "batch_users").Str("userID", userID).Err(err).Msg("slack API error for user")

				// Try SQLite cache as fallback
				cached, cacheErr := h.store.GetCachedUser(userID)
				if cacheErr == nil && cached != nil && cached.RealName != "" {
					results[userID] = map[string]string{
						"real_name":     cached.RealName,
						"profile_image": cached.ProfileImage,
					}
				}
				continue
			}

			if user.RealName != "" {
				results[userID] = map[string]string{
					"real_name":     user.RealName,
					"profile_image": user.Profile.Image192,
				}

				// Prepare for caching
				toCache[userID] = UserCacheData{
					RealName:     user.RealName,
					ProfileImage: user.Profile.Image192,
				}

				// Cache to SQLite
				if err := h.store.SetCachedUser(userID, user.RealName, user.Profile.Image192); err != nil {
					h.logger.Warn().Str("handler", "batch_users").Str("userID", userID).Err(err).Msg("failed to cache user to sqlite")
				}
			} else {
				// Try SQLite cache for deactivated users
				cached, cacheErr := h.store.GetCachedUser(userID)
				if cacheErr == nil && cached != nil && cached.RealName != "" {
					results[userID] = map[string]string{
						"real_name":     cached.RealName,
						"profile_image": cached.ProfileImage,
					}
				}
			}
		}

		// Batch cache to Redis
		if h.redisCache != nil && len(toCache) > 0 {
			if err := h.redisCache.SetUsers(ctx, toCache); err != nil {
				h.logger.Warn().Str("handler", "batch_users").Err(err).Msg("failed to batch cache users to redis")
			}
		}
	}

	h.logger.Info().Str("handler", "batch_users").Int("requested", len(userIDs)).Int("found", len(results)).Msg("request completed")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		h.logger.Error().Str("handler", "batch_users").Err(err).Msg("failed to encode response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
