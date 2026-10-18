package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"regexp"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// EventProcessor handles Slack event processing
type EventProcessor struct {
	store         *SQLiteStore
	slackManager  *SlackConnectionManager
	redisCache    *RedisUserCache
	channelID     string
	emoji         string
	maxPerDay     int
	logger        zerolog.Logger
	mentionRe     *regexp.Regexp
	emojiRe       *regexp.Regexp
	msgsProcessed *prometheus.CounterVec

	// Worker pool (see StartWorkers). Without it, events are processed inline.
	poolMu sync.RWMutex
	shards []chan func()
	closed bool
	wg     sync.WaitGroup
}

// shardQueueSize is the number of queued events per worker before HandleEvent blocks
// (back-pressure towards the socket mode event loop, same as the former inline processing).
const shardQueueSize = 64

// StartWorkers starts n workers that run the side effects of beer messages (daily limit check,
// database writes, Redis update, Slack replies). An event is assigned to a worker by giver, so
// the events of one user keep their order and the daily limit check cannot race with the
// writes of the same user, while different users are processed in parallel. Call it once,
// before the first event; n < 1 keeps the inline processing.
func (ep *EventProcessor) StartWorkers(n int) {
	if n < 1 {
		return
	}
	ep.poolMu.Lock()
	defer ep.poolMu.Unlock()
	if ep.shards != nil || ep.closed {
		return
	}
	ep.shards = make([]chan func(), n)
	for i := range ep.shards {
		queue := make(chan func(), shardQueueSize)
		ep.shards[i] = queue
		ep.wg.Add(1)
		go func() {
			defer ep.wg.Done()
			for job := range queue {
				ep.runJob(job)
			}
		}()
	}
}

// runJob runs one job and keeps the worker alive if it panics.
func (ep *EventProcessor) runJob(job func()) {
	defer func() {
		if r := recover(); r != nil {
			ep.logger.Error().Interface("panic", r).Msg("event worker recovered from panic")
		}
	}()
	job()
}

// dispatch runs job on the worker for giver, or inline if no pool is running (not started or
// already shut down).
func (ep *EventProcessor) dispatch(giver string, job func()) {
	ep.poolMu.RLock()
	if !ep.closed && len(ep.shards) > 0 {
		ep.shards[ep.shardIndex(giver)] <- job
		ep.poolMu.RUnlock()
		return
	}
	ep.poolMu.RUnlock()
	job()
}

// shardIndex returns the worker (queue) responsible for a giver.
func (ep *EventProcessor) shardIndex(giver string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(giver))
	return int(h.Sum32() % uint32(len(ep.shards)))
}

// Shutdown stops accepting queued work, lets the workers finish everything already queued and
// returns when they are done or ctx expires. Events that arrive afterwards run inline.
func (ep *EventProcessor) Shutdown(ctx context.Context) error {
	ep.poolMu.Lock()
	if !ep.closed {
		ep.closed = true
		for _, queue := range ep.shards {
			close(queue)
		}
	}
	ep.poolMu.Unlock()

	done := make(chan struct{})
	go func() {
		ep.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NewEventProcessor creates a new EventProcessor
func NewEventProcessor(store *SQLiteStore, slackManager *SlackConnectionManager, redisCache *RedisUserCache, channelID, emoji string, maxPerDay int, logger zerolog.Logger, msgsProcessed *prometheus.CounterVec) *EventProcessor {
	return &EventProcessor{
		store:         store,
		slackManager:  slackManager,
		redisCache:    redisCache,
		channelID:     channelID,
		emoji:         emoji,
		maxPerDay:     maxPerDay,
		logger:        logger,
		mentionRe:     regexp.MustCompile(`<@([A-Z0-9]+)>`),
		emojiRe:       regexp.MustCompile(regexp.QuoteMeta(emoji)),
		msgsProcessed: msgsProcessed,
	}
}

// HandleEvent processes a socketmode event
func (ep *EventProcessor) HandleEvent(evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		if evt.Request == nil {
			ep.logger.Warn().Msg("received EventTypeEventsAPI with nil request")
			return
		}
		if socketClient := ep.slackManager.GetSocketClient(); socketClient != nil {
			// Since slack-go v0.21 Ack returns an error (response >= 20KB or cancelled
			// context). An empty ack cannot hit either, but log it instead of dropping it.
			// Processing continues as before: a lost ack only causes a redelivery, which
			// the deduplication below handles.
			if err := socketClient.Ack(*evt.Request); err != nil {
				ep.logger.Error().Err(err).Str("envelopeID", evt.Request.EnvelopeID).Msg("failed to ack event")
			}
		} else {
			ep.logger.Warn().Msg("socket client is nil, cannot ack event")
			return
		}
		eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			ep.logger.Warn().Str("type", fmt.Sprintf("%T", evt.Data)).Msg("unexpected event data type")
			return
		}
		// Deduplicate based on the Events API envelope ID when available.
		// Try to get a stable envelope id from the socketmode event request
		envelopeID := "" //nolint:typecheck // Used in event ID generation below
		if evt.Request.EnvelopeID != "" {
			envelopeID = evt.Request.EnvelopeID
		}
		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			inner := eventsAPIEvent.InnerEvent
			switch ev := inner.Data.(type) {
			case *slackevents.MessageEvent:
				ep.handleMessageEvent(ev, envelopeID)
			default:
				// ignore non-message events
			}
		}
	default:
		// Handle other event types if needed
	}
}

// handleMessageEvent checks a Slack message event, marks it as processed and hands the beer
// handling to the worker pool.
//
// Deduplication stays synchronous and happens before any side effect: the event is marked in
// TryMarkEventProcessed on the event loop, and only the first delivery is dispatched.
func (ep *EventProcessor) handleMessageEvent(ev *slackevents.MessageEvent, envelopeID string) {
	// limit to the configured channel and only user messages
	if ev.Channel != ep.channelID || ev.User == "" {
		return
	}
	// ignore message subtypes (edits, bot messages, etc.) -- only plain messages
	// SubType is empty for normal user messages
	if ev.SubType != "" {
		return
	}

	// compute a stable event id for message events: prefer envelopeID if present,
	// otherwise build one from channel|user|ts which is stable across redeliveries
	eventID := envelopeID
	if eventID == "" {
		eventID = fmt.Sprintf("msg|%s|%s|%s", ev.Channel, ev.User, ev.TimeStamp)
	}
	// Attempt to mark the event as processed before doing work.
	// INSERT OR IGNORE will return 0 affected rows if the event
	// already exists; in that case we skip processing. This
	// avoids the race where two deliveries check IsEventProcessed
	// concurrently and both proceed to write/Log.
	if eventID != "" {
		if ok, err := ep.store.TryMarkEventProcessed(eventID, time.Now()); err != nil {
			ep.logger.Error().Err(err).Str("eventID", eventID).Msg("failed to try-mark event processed")
			return
		} else if !ok {
			ep.logger.Debug().Str("eventID", eventID).Msg("event already processed, skipping")
			return
		}
	}
	ep.logger.Debug().Str("eventID", eventID).Str("user", ev.User).Str("channel", ev.Channel).Msg("processing message event")

	// Increment Prometheus counter
	if ep.msgsProcessed != nil {
		ep.msgsProcessed.WithLabelValues(ev.Channel).Inc()
	}

	// New logic: associate beers with the last seen mention
	mentions := ep.mentionRe.FindAllStringSubmatch(ev.Text, -1)
	mentionIndices := ep.mentionRe.FindAllStringSubmatchIndex(ev.Text, -1)
	emojiIndices := ep.emojiRe.FindAllStringIndex(ev.Text, -1)

	if len(mentions) == 0 || len(emojiIndices) == 0 {
		return
	}

	recipientBeers := make(map[string]int)
	for _, emojiIdx := range emojiIndices {
		lastMentionIdx := -1
		var recipientID string
		for i, mentionIdx := range mentionIndices {
			if emojiIdx[0] > mentionIdx[1] { // emoji is after mention
				if mentionIdx[0] > lastMentionIdx {
					lastMentionIdx = mentionIdx[0]
					recipientID = mentions[i][1]
				}
			}
		}
		// Prevent self-gifting
		if recipientID != "" && recipientID != ev.User {
			recipientBeers[recipientID]++
		}
	}

	totalBeersToGive := 0
	for _, count := range recipientBeers {
		totalBeersToGive += count
	}

	if totalBeersToGive == 0 {
		return
	}

	ep.dispatch(ev.User, func() { ep.giveBeers(ev, recipientBeers, totalBeersToGive) })
}

// giveBeers enforces the daily limit, stores the beers, updates Redis and confirms in the
// channel. The event is already marked as processed (see handleMessageEvent).
func (ep *EventProcessor) giveBeers(ev *slackevents.MessageEvent, recipientBeers map[string]int, totalBeersToGive int) {
	client := ep.slackManager.GetClient()
	today := time.Now().UTC().Format("2006-01-02")
	givenToday, err := ep.store.CountGivenOnDate(ev.User, today)
	if err != nil {
		ep.logger.Error().Err(err).Str("user", ev.User).Str("date", today).Msg("count given on date failed")
		return
	}

	if givenToday >= ep.maxPerDay {
		ep.logger.Info().Str("user", ev.User).Int("givenToday", givenToday).Int("maxPerDay", ep.maxPerDay).Msg("daily limit reached")
		message := fmt.Sprintf("Sorry <@%s>, you have reached your daily limit of %d beers.", ev.User, ep.maxPerDay)
		if _, _, err := client.PostMessage(ev.Channel, slack.MsgOptionText(message, false)); err != nil {
			ep.logger.Error().Err(err).Str("channel", ev.Channel).Msg("failed to post daily limit message")
		}
		return
	}

	allowed := ep.maxPerDay - givenToday
	if totalBeersToGive > allowed {
		ep.logger.Info().Str("user", ev.User).Int("givenToday", givenToday).Int("totalBeersToGive", totalBeersToGive).Int("allowed", allowed).Msg("daily limit would be exceeded")
		message := fmt.Sprintf("Sorry <@%s>, you are trying to give %d beers, but you only have %d left for today.", ev.User, totalBeersToGive, allowed)
		if _, _, err := client.PostMessage(ev.Channel, slack.MsgOptionText(message, false)); err != nil {
			ep.logger.Error().Err(err).Str("channel", ev.Channel).Msg("failed to post limit exceeded message")
		}
		return
	}

	for recipient, count := range recipientBeers {
		var eventTime time.Time
		if ev.TimeStamp != "" {
			if t, err := parseSlackTimestamp(ev.TimeStamp); err == nil {
				eventTime = t
			} else {
				ep.logger.Warn().Err(err).Str("timestamp", ev.TimeStamp).Msg("failed to parse slack timestamp, using current time")
				eventTime = time.Now()
			}
		} else {
			eventTime = time.Now()
		}
		if err := ep.store.AddBeer(ev.User, recipient, ev.TimeStamp, eventTime, count); err != nil {
			ep.logger.Error().Err(err).Str("giver", ev.User).Str("recipient", recipient).Int("count", count).Msg("failed to add beer")
		} else {
			ep.logger.Info().Str("giver", ev.User).Str("recipient", recipient).Int("count", count).Msg("beer given")

			// Update Redis beer stats (write-through cache)
			if ep.redisCache != nil {
				ctx := context.Background()

				// Increment beer stats in Redis (write-through cache)
				if err := ep.redisCache.IncrementGivenStatsAt(ctx, ev.User, count, eventTime); err != nil {
					ep.logger.Warn().Err(err).Str("giver", ev.User).Int("count", count).Msg("failed to increment given stats in redis")
				}
				if err := ep.redisCache.IncrementReceivedStatsAt(ctx, recipient, count, eventTime); err != nil {
					ep.logger.Warn().Err(err).Str("recipient", recipient).Int("count", count).Msg("failed to increment received stats in redis")
				}
			}

			// Post confirmation message to channel
			var message string
			if count == 1 {
				message = fmt.Sprintf("<@%s> gave 1 beer to <@%s>!", ev.User, recipient)
			} else {
				message = fmt.Sprintf("<@%s> gave %d beers to <@%s>!", ev.User, count, recipient)
			}
			if _, _, err := client.PostMessage(ev.Channel, slack.MsgOptionText(message, false)); err != nil {
				ep.logger.Error().Err(err).Str("channel", ev.Channel).Msg("failed to post beer confirmation message")
			}
		}
	}
}
