package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// These tests feed raw Events API payloads through the same parser socketmode uses
// (slackevents.ParseEvent with OptionNoVerifyToken) into EventProcessor.HandleEvent.
// They pin the slack-go types the bot relies on (EventsAPIEvent value in evt.Data,
// *MessageEvent as inner data, Request.EnvelopeID) and run without network access:
// the Web API is a local httptest server.

const (
	testChannel   = "C0TESTCHAN"
	testGiver     = "UTESTGIVER"
	testRecipient = "UTESTRECV"
)

// fakeSlackAPI records chat.postMessage calls.
type fakeSlackAPI struct {
	mu       sync.Mutex
	messages []string
}

func (f *fakeSlackAPI) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != "/chat.postMessage" {
		_, _ = w.Write([]byte(`{"ok":false,"error":"unknown_method"}`))
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.messages = append(f.messages, r.PostForm.Get("text"))
	f.mu.Unlock()
	_, _ = fmt.Fprintf(w, `{"ok":true,"channel":%q,"ts":"1700000000.000200"}`, r.PostForm.Get("channel"))
}

func (f *fakeSlackAPI) posted() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.messages...)
}

func newTestEventProcessor(t *testing.T) (*EventProcessor, *SQLiteStore, *fakeSlackAPI) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "events.db")+"?_foreign_keys=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	fake := &fakeSlackAPI{}
	srv := httptest.NewServer(http.HandlerFunc(fake.handler))
	t.Cleanup(srv.Close)

	api := slack.New("xoxb-test", slack.OptionAPIURL(srv.URL+"/"))
	manager := &SlackConnectionManager{
		client:       api,
		socketClient: socketmode.New(api), // Ack only queues the response (buffered channel)
		logger:       zerolog.Nop(),
	}
	ep := NewEventProcessor(store, manager, nil, testChannel, ":beer:", 10, zerolog.Nop(), nil)
	return ep, store, fake
}

// socketmodeEvent builds the socketmode.Event the client emits for an events_api envelope.
func socketmodeEvent(t *testing.T, envelopeID string, inner map[string]any) socketmode.Event {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"token":      "test-token",
		"team_id":    "T0TEST",
		"api_app_id": "A0TEST",
		"type":       "event_callback",
		"event_id":   "Ev0TEST",
		"event_time": time.Now().Unix(),
		"event":      inner,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	parsed, err := slackevents.ParseEvent(payload, slackevents.OptionNoVerifyToken())
	if err != nil {
		t.Fatalf("parse event: %v", err)
	}
	return socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: parsed,
		Request: &socketmode.Request{
			Type:       socketmode.RequestTypeEventsAPI,
			EnvelopeID: envelopeID,
			Payload:    payload,
		},
	}
}

func messageEvent(channel, user, text, ts string) map[string]any {
	return map[string]any{
		"type":         "message",
		"channel":      channel,
		"channel_type": "channel",
		"user":         user,
		"text":         text,
		"ts":           ts,
		"event_ts":     ts,
	}
}

func TestHandleEvent_SocketModeMessageGivesBeers(t *testing.T) {
	ep, store, fake := newTestEventProcessor(t)

	now := time.Now().UTC()
	ts := fmt.Sprintf("%d.000100", now.Unix())
	evt := socketmodeEvent(t, "env-1", messageEvent(testChannel, testGiver, fmt.Sprintf("thanks <@%s> :beer: :beer:", testRecipient), ts))

	// The parsed data must still be the value type the handler asserts on.
	apiEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		t.Fatalf("evt.Data is %T, want slackevents.EventsAPIEvent", evt.Data)
	}
	if _, ok := apiEvent.InnerEvent.Data.(*slackevents.MessageEvent); !ok {
		t.Fatalf("inner data is %T, want *slackevents.MessageEvent", apiEvent.InnerEvent.Data)
	}

	ep.HandleEvent(evt)

	given, err := store.CountGivenOnDate(testGiver, now.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("count given: %v", err)
	}
	if given != 2 {
		t.Fatalf("given = %d, want 2", given)
	}
	want := fmt.Sprintf("<@%s> gave 2 beers to <@%s>!", testGiver, testRecipient)
	if got := fake.posted(); len(got) != 1 || got[0] != want {
		t.Fatalf("posted = %q, want [%q]", got, want)
	}

	// A redelivery of the same envelope is deduplicated: no new beers, no new message.
	ep.HandleEvent(socketmodeEvent(t, "env-1", messageEvent(testChannel, testGiver, fmt.Sprintf("thanks <@%s> :beer: :beer:", testRecipient), ts)))
	if given, _ := store.CountGivenOnDate(testGiver, now.Format("2006-01-02")); given != 2 {
		t.Fatalf("after redelivery given = %d, want 2", given)
	}
	if got := fake.posted(); len(got) != 1 {
		t.Fatalf("after redelivery posted %d messages, want 1", len(got))
	}
}

func TestHandleEvent_SocketModeIgnoredMessages(t *testing.T) {
	ts := fmt.Sprintf("%d.000100", time.Now().Unix())
	text := fmt.Sprintf("<@%s> :beer:", testRecipient)

	edited := messageEvent(testChannel, testGiver, text, ts)
	edited["subtype"] = "message_changed"

	cases := map[string]map[string]any{
		"other channel": messageEvent("C0OTHER", testGiver, text, ts),
		"subtype":       edited,
		"self gift":     messageEvent(testChannel, testGiver, fmt.Sprintf("<@%s> :beer:", testGiver), ts),
		"no emoji":      messageEvent(testChannel, testGiver, fmt.Sprintf("<@%s> thanks", testRecipient), ts),
	}
	for name, inner := range cases {
		t.Run(name, func(t *testing.T) {
			ep, store, fake := newTestEventProcessor(t)
			ep.HandleEvent(socketmodeEvent(t, "env-"+name, inner))

			givers, err := store.GetAllGivers()
			if err != nil {
				t.Fatalf("get givers: %v", err)
			}
			if len(givers) != 0 {
				t.Fatalf("givers = %v, want none", givers)
			}
			if got := fake.posted(); len(got) != 0 {
				t.Fatalf("posted = %q, want none", got)
			}
		})
	}
}
