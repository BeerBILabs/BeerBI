package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	emoji := ":beer:" //nolint:typecheck // Used in regexp compilation below
	if env := os.Getenv("EMOJI"); env != "" {
		emoji = env
	}
	dbPath := flag.String("db", os.Getenv("DB_PATH"), "sqlite database path")
	botToken := flag.String("bot-token", os.Getenv("BOT_TOKEN"), "slack bot token (xoxb-...)")
	appToken := flag.String("app-token", os.Getenv("APP_TOKEN"), "slack app-level token (xapp-...)")
	channelID := flag.String("channel", os.Getenv("CHANNEL"), "channel id to monitor")
	apiToken := flag.String("api-token", os.Getenv("API_TOKEN"), "api token for authentication")
	logLevel := flag.String("log-level", os.Getenv("LOG_LEVEL"), "log level")

	addrDefault := ":8080"
	if env := os.Getenv("ADDR"); env != "" {
		addrDefault = env
	}
	addr := flag.String("addr", addrDefault, "health/metrics listen address")

	maxPerDayDefault := 10
	if env := os.Getenv("MAX_PER_DAY"); env != "" {
		if v, err := strconv.Atoi(env); err == nil {
			maxPerDayDefault = v
		}
	}
	maxPerDay := flag.Int("max-per-day", maxPerDayDefault, "max beers a user may give per day") //nolint:typecheck // Used in daily limit checks
	eventWorkersDefault := defaultEventWorkers
	if env := os.Getenv("EVENT_WORKERS"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v >= 0 {
			eventWorkersDefault = v
		}
	}
	eventWorkers := flag.Int("event-workers", eventWorkersDefault, "workers processing beer messages in parallel (0 = inline)")

	retentionFlag := flag.String("processed-events-retention-days", os.Getenv("PROCESSED_EVENTS_RETENTION_DAYS"), "days to keep Slack event deduplication records (processed_events), 1-3650, default 30")
	flag.Parse()
	// Validated once, after the flag (which defaults to the env value) is known.
	retentionDays, err := ParseRetentionDays(*retentionFlag)
	if err != nil {
		log.Fatalf("processed-events-retention-days / PROCESSED_EVENTS_RETENTION_DAYS: %v", err)
	}

	if *botToken == "" || *appToken == "" || *channelID == "" {
		log.Fatal("bot-token, app-token and channel must be provided via flags or env (BOT_TOKEN, APP_TOKEN, CHANNEL)")
	}

	// structured logger (zerolog)
	zerolog.TimeFieldFormat = time.RFC3339
	zlogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	// Set log level from flag
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel // default to info if invalid/empty
	}
	zlogger = zlogger.Level(level)
	zlog.Logger = zlogger

	// open sqlite (the logger is set up first so migration messages use it)
	dbDir := ""
	dbFile := *dbPath
	if idx := strings.LastIndex(dbFile, "/"); idx != -1 {
		dbDir = dbFile[:idx]
	}
	if dbDir != "" {
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			if err := os.MkdirAll(dbDir, 0o755); err != nil {
				log.Fatalf("Failed to create DB directory: %v", err)
			}
		}
	}
	db, err := openDB(dbFile)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	store, err := NewSQLiteStore(db)
	if err != nil {
		_ = db.Close()
		log.Fatalf("init store: %v", err)
	}
	// Runs PRAGMA optimize before closing (deferred calls run on return from
	// main, after the graceful shutdown below).
	defer func() {
		if err := store.Close(); err != nil {
			zlog.Error().Err(err).Msg("close db")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis cache
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "cache:6379" // Default to Docker service name
	}
	redisCache, err := NewRedisUserCache(redisAddr, zlogger)
	if err != nil {
		zlogger.Warn().Err(err).Msg("redis connection failed, continuing without cache")
		redisCache = nil // Graceful fallback - continue without Redis
	} else {
		defer func() { _ = redisCache.Close() }()
	}

	// init Slack connection manager
	slackManager := NewSlackConnectionManager(*botToken, *appToken, zlogger)

	// Prometheus metrics
	msgsProcessed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bwm_messages_processed_total",
		Help: "Number of messages processed by the bot",
	}, []string{"channel"})
	prometheus.MustRegister(msgsProcessed)

	// Setup HTTP handlers
	handlers := NewAPIHandlers(store, slackManager.GetClient(), slackManager, redisCache, zlogger)

	if *apiToken == "" {
		zlogger.Error().Msg("API_TOKEN is not set: every endpoint below /api except /api/health will answer 401")
	}
	srv := newHTTPServer(*addr, newRouter(handlers, *apiToken, zlogger))
	go func() {
		zlogger.Info().Str("addr", *addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zlogger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create event processor and handler
	eventProcessor := NewEventProcessor(store, slackManager, redisCache, *channelID, emoji, *maxPerDay, zlogger, msgsProcessed)
	eventProcessor.StartWorkers(*eventWorkers)

	// Start Slack connection manager with automatic reconnection
	slackManager.StartWithReconnection(ctx, eventProcessor.HandleEvent)

	// Prune Slack event deduplication records older than the retention, daily.
	zlogger.Info().Int("retention_days", retentionDays).Msg("processed_events pruning enabled")
	go store.RunProcessedEventsPruner(ctx, 24*time.Hour, retentionDays)

	// Refresh planner statistics where needed (PRAGMA optimize), hourly.
	go store.RunOptimizer(ctx, time.Hour)

	// Start Redis sync worker (if Redis is available)
	if redisCache != nil {
		go redisCache.StartSyncWorker(ctx, store, 5*time.Minute)
	}

	// Preload user names from users.list now and daily, so /api/users and the
	// inline names in the stats responses rarely need a users.info call.
	go handlers.users.RunWarmup(ctx, 24*time.Hour)

	// Connection health monitor
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				connected := slackManager.IsConnected()
				if !connected {
					zlogger.Warn().Msg("Slack connection monitor: DISCONNECTED")
				} else {
					// Test actual API connection periodically
					if err := slackManager.TestConnection(ctx); err != nil {
						zlogger.Error().Err(err).Msg("Slack connection monitor: API test failed")
					}
				}
			case <-ctx.Done():
				zlogger.Info().Msg("Connection monitor stopping")
				return
			}
		}
	}()

	select {
	case sig := <-sigs:
		zlogger.Info().Str("signal", sig.String()).Msg("received shutdown signal")
		cancel() // Cancel context to stop socketmode
	case <-ctx.Done():
		zlogger.Info().Msg("context cancelled, shutting down")
	}

	// Graceful shutdown. Order matters: the Slack client stopped with cancel() above (no new
	// events), the workers finish the queued beer messages, then the HTTP server drains its
	// requests; the deferred Close calls of the database and Redis run last.
	zlogger.Info().Msg("initiating graceful shutdown")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := eventProcessor.Shutdown(ctxShutdown); err != nil {
		zlogger.Error().Err(err).Msg("event workers did not finish in time")
	} else {
		zlogger.Info().Msg("event workers drained")
	}
	if err := srv.Shutdown(ctxShutdown); err != nil {
		zlogger.Error().Err(err).Msg("HTTP server shutdown error")
	} else {
		zlogger.Info().Msg("HTTP server shutdown completed")
	}
	zlogger.Info().Msg("shutdown complete")
}

// HTTP server limits. WriteTimeout bounds the slowest endpoint (the combined analytics query)
// with a wide margin; the other values stop slow or idle clients from holding connections.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 64 << 10
	// shutdownTimeout is the total time for draining event workers and HTTP requests.
	shutdownTimeout = 15 * time.Second
	// defaultEventWorkers is the default number of parallel event workers (EVENT_WORKERS).
	defaultEventWorkers = 4
)

// newHTTPServer returns the HTTP server with timeouts and a header size limit.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

// newRouter registers all routes. Everything below /api except /api/health needs the bearer token.
func newRouter(handlers *APIHandlers, apiToken string, logger zerolog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", promhttp.Handler())

	authed := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, authMiddleware(apiToken, logger, h))
	}

	// REST API endpoints (auth required)
	authed("/api/given", handlers.GivenHandler)
	authed("/api/received", handlers.ReceivedHandler)
	authed("/api/user", handlers.UserHandler)
	authed("/api/users", handlers.BatchUsersHandler)
	mux.HandleFunc("/api/health", handlers.HealthHandler)

	// Stats/Analytics endpoints (auth required)
	authed("/api/stats/combined", handlers.CombinedAnalyticsHandler)
	authed("/api/stats/timeline", handlers.TimelineHandler)
	authed("/api/stats/quarterly", handlers.QuarterlyHandler)
	authed("/api/stats/top", handlers.TopUsersHandler)
	authed("/api/stats/heatmap", handlers.HeatmapHandler)
	authed("/api/stats/pairs", handlers.PairsHandler)
	return mux
}
