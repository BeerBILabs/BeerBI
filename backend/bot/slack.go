package main

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SlackConnectionManager handles Slack socket mode connection with reconnection logic
type SlackConnectionManager struct {
	client         *slack.Client
	socketClient   *socketmode.Client
	botToken       string
	appToken       string
	isConnected    bool
	lastPing       time.Time
	reconnectCount int
	mu             sync.RWMutex
	logger         zerolog.Logger
}

// NewSlackConnectionManager creates a new connection manager
func NewSlackConnectionManager(botToken, appToken string, logger zerolog.Logger) *SlackConnectionManager {
	client := slack.New(botToken, slack.OptionAppLevelToken(appToken))
	socketClient := socketmode.New(client)

	return &SlackConnectionManager{
		client:       client,
		socketClient: socketClient,
		botToken:     botToken,
		appToken:     appToken,
		lastPing:     time.Now(),
		logger:       logger,
	}
}

// IsConnected returns the current connection status
func (scm *SlackConnectionManager) IsConnected() bool {
	scm.mu.RLock()
	defer scm.mu.RUnlock()
	return scm.isConnected
}

// SetConnected updates the connection status
func (scm *SlackConnectionManager) setConnected(connected bool) {
	scm.mu.Lock()
	defer scm.mu.Unlock()
	scm.isConnected = connected
	if connected {
		scm.lastPing = time.Now()
		scm.logger.Info().Int("reconnectCount", scm.reconnectCount).Msg("Slack connection established")
	} else {
		scm.logger.Warn().Msg("Slack connection lost")
	}
}

// GetClient returns the Slack client
func (scm *SlackConnectionManager) GetClient() *slack.Client {
	return scm.client
}

// GetSocketClient returns the socket mode client
func (scm *SlackConnectionManager) GetSocketClient() *socketmode.Client {
	scm.mu.RLock()
	defer scm.mu.RUnlock()
	return scm.socketClient
}

// TestConnection tests the Slack API connection
func (scm *SlackConnectionManager) TestConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := scm.client.AuthTestContext(ctx)
	return err
}

// StartWithReconnection starts the socket mode client with automatic reconnection
func (scm *SlackConnectionManager) StartWithReconnection(ctx context.Context, eventHandler func(socketmode.Event)) {
	const maxReconnectDelay = 5 * time.Minute

	go func() {
		for {
			select {
			case <-ctx.Done():
				scm.logger.Info().Msg("Connection manager shutting down")
				return
			default:
				// Calculate exponential backoff delay
				delay := time.Duration(math.Pow(2, float64(scm.reconnectCount))) * time.Second
				if delay > maxReconnectDelay {
					delay = maxReconnectDelay
				}

				if scm.reconnectCount > 0 {
					scm.logger.Info().Dur("delay", delay).Int("attempt", scm.reconnectCount+1).Msg("Reconnecting to Slack")
					select {
					case <-time.After(delay):
					case <-ctx.Done():
						return
					}
				}

				// Test connection first
				if err := scm.TestConnection(ctx); err != nil {
					scm.logger.Error().Err(err).Msg("Slack API connection test failed")
					scm.reconnectCount++
					continue
				}

				// Create new socket client for this connection attempt
				scm.mu.Lock()
				scm.socketClient = socketmode.New(scm.client)
				scm.mu.Unlock()
				scm.reconnectCount = 0

				// Start event processing
				go scm.processEvents(eventHandler)

				// Run the socket mode client
				scm.logger.Info().Msg("Starting Slack socket mode client")
				if err := scm.socketClient.RunContext(ctx); err != nil {
					scm.setConnected(false)
					if ctx.Err() != nil {
						scm.logger.Info().Err(err).Msg("Socket mode client stopped due to context cancellation")
						return
					} else {
						scm.logger.Error().Err(err).Msg("Socket mode client error, will reconnect")
						scm.reconnectCount++
					}
				} else {
					scm.setConnected(false)
					scm.logger.Info().Msg("Socket mode client stopped gracefully")
				}
			}
		}
	}()
}

// processEvents handles socket mode events
func (scm *SlackConnectionManager) processEvents(eventHandler func(socketmode.Event)) {
	for evt := range scm.socketClient.Events {
		scm.mu.Lock()
		scm.lastPing = time.Now()
		scm.mu.Unlock()

		// Handle special events
		if evt.Type == socketmode.EventTypeHello {
			scm.logger.Debug().Msg("Slack socket mode: hello received")
			scm.setConnected(true)
		}

		// Call the custom event handler
		if eventHandler != nil {
			eventHandler(evt)
		}
	}
}
