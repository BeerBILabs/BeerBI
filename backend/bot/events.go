package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// EventProcessor handles Slack event processing
type EventProcessor struct {
	store        *SQLiteStore
	slackManager *SlackConnectionManager
	channelID    string
	emoji        string
	maxPerDay    int
	logger       zerolog.Logger
}

// NewEventProcessor creates a new EventProcessor
func NewEventProcessor(store *SQLiteStore, slackManager *SlackConnectionManager, channelID, emoji string, maxPerDay int, logger zerolog.Logger) *EventProcessor {
	return &EventProcessor{
		store:        store,
		slackManager: slackManager,
		channelID:    channelID,
		emoji:        emoji,
		maxPerDay:    maxPerDay,
		logger:       logger,
	}
}

// HandleEvent processes a socketmode event
func (ep *EventProcessor) HandleEvent(evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		ep.slackManager.GetSocketClient().Ack(*evt.Request)
		eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			ep.logger.Warn().Str("type", fmt.Sprintf("%T", evt.Data)).Msg("unexpected event data type")
			return
		}
		// Deduplicate based on the Events API envelope ID when available.
		// Try to get a stable envelope id from the socketmode event request
		envelopeID := "" //nolint:typecheck // Used in event ID generation below
		if evt.Request != nil && evt.Request.EnvelopeID != "" {
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

// handleMessageEvent processes a Slack message event
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
	// New logic: associate beers with the last seen mention

	mentionRe := regexp.MustCompile(`<@([A-Z0-9]+)>`)
	emojiRe := regexp.MustCompile(regexp.QuoteMeta(ep.emoji))

	mentions := mentionRe.FindAllStringSubmatch(ev.Text, -1)
	mentionIndices := mentionRe.FindAllStringSubmatchIndex(ev.Text, -1)
	emojiIndices := emojiRe.FindAllStringIndex(ev.Text, -1)

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
		if recipientID != "" {
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
		}
		ep.logger.Info().Str("giver", ev.User).Str("recipient", recipient).Int("count", count).Msg("beer given")
	}
	// event was pre-marked via TryMarkEventProcessed
}
