package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	flag.Parse()

	if *botToken == "" || *appToken == "" || *channelID == "" {
		log.Fatal("bot-token, app-token and channel must be provided via flags or env (BOT_TOKEN, APP_TOKEN, CHANNEL)")
	}

	// open sqlite
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
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewSQLiteStore(db)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		defer redisCache.Close()
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

	// HTTP server for health + metrics
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", promhttp.Handler())

	// REST API endpoints
	mux.Handle("/api/given", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.GivenHandler)))
	mux.Handle("/api/received", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.ReceivedHandler)))
	mux.Handle("/api/user", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.UserHandler)))
	mux.Handle("/api/users", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.BatchUsersHandler)))
	// Public endpoints (no auth required)
	mux.Handle("/api/givers", http.HandlerFunc(handlers.GiversHandler))
	mux.Handle("/api/recipients", http.HandlerFunc(handlers.RecipientsHandler))
	mux.HandleFunc("/api/health", handlers.HealthHandler)

	// Stats/Analytics endpoints (auth required)
	mux.Handle("/api/stats/timeline", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.TimelineHandler)))
	mux.Handle("/api/stats/quarterly", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.QuarterlyHandler)))
	mux.Handle("/api/stats/top", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.TopUsersHandler)))
	mux.Handle("/api/stats/heatmap", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.HeatmapHandler)))
	mux.Handle("/api/stats/pairs", authMiddleware(*apiToken, zlogger, http.HandlerFunc(handlers.PairsHandler)))

	srv := &http.Server{Addr: *addr, Handler: mux}
	go func() {
		zlogger.Info().Str("addr", *addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlogger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create event processor and handler
	eventProcessor := NewEventProcessor(store, slackManager, redisCache, *channelID, emoji, *maxPerDay, zlogger, msgsProcessed)

	// Start Slack connection manager with automatic reconnection
	slackManager.StartWithReconnection(ctx, eventProcessor.HandleEvent)

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

	// shutdown http
	zlogger.Info().Msg("initiating graceful shutdown")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		zlogger.Error().Err(err).Msg("HTTP server shutdown error")
	} else {
		zlogger.Info().Msg("HTTP server shutdown completed")
	}
	zlogger.Info().Msg("shutdown complete")
	// socketmode client will stop when context is cancelled / RunContext returns
}
