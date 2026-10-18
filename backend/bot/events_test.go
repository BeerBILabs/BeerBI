package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// These tests call handleMessageEvent directly (many events, no socket mode ack) on top of the
// store and fake Slack API from newTestEventProcessor (events_socketmode_test.go).

func todayUTC() string { return time.Now().UTC().Format(dateLayout) }

var tsCounter atomic.Int64

// uniqueTS returns a Slack timestamp for now that differs on every call.
func uniqueTS() string {
	return fmt.Sprintf("%d.%06d", time.Now().Unix(), tsCounter.Add(1))
}

func msgEvent(channel, user, text string) *slackevents.MessageEvent {
	return &slackevents.MessageEvent{Channel: channel, User: user, Text: text, TimeStamp: uniqueTS()}
}

func mention(id string) string { return "<@" + id + ">" }

func receivedToday(t *testing.T, store *SQLiteStore, user string) int {
	t.Helper()
	n, err := store.CountReceived(user, todayUTC())
	if err != nil {
		t.Fatalf("count received: %v", err)
	}
	return n
}

func givenToday(t *testing.T, store *SQLiteStore, user string) int {
	t.Helper()
	n, err := store.CountGivenOnDate(user, todayUTC())
	if err != nil {
		t.Fatalf("count given: %v", err)
	}
	return n
}

func TestHandleMessageEventAssignsBeersToLastMention(t *testing.T) {
	const recvC = "URECVCCCC"
	cases := []struct {
		name string
		text string
		want map[string]int
	}{
		{"one mention one beer", mention(testRecipient) + " :beer:", map[string]int{testRecipient: 1}},
		{"two beers for one mention", mention(testRecipient) + " :beer: :beer:", map[string]int{testRecipient: 2}},
		{"last mention before the emoji wins", mention(testRecipient) + " " + mention(recvC) + " :beer:", map[string]int{recvC: 1}},
		{"beer after each mention", mention(testRecipient) + " :beer: " + mention(recvC) + " :beer: :beer:", map[string]int{testRecipient: 1, recvC: 2}},
		{"emoji before the mention counts for nobody", ":beer: " + mention(testRecipient), map[string]int{}},
		{"mention without emoji", mention(testRecipient) + " thanks", map[string]int{}},
		{"emoji without mention", "cheers :beer:", map[string]int{}},
		{"self mention is skipped, next recipient still counts", mention(testGiver) + " :beer: " + mention(testRecipient) + " :beer:", map[string]int{testRecipient: 1}},
		{"other emoji", mention(testRecipient) + " :wine_glass:", map[string]int{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ep, store, fake := newTestEventProcessor(t)
			ep.handleMessageEvent(msgEvent(testChannel, testGiver, tc.text), "")

			total := 0
			for _, recipient := range []string{testRecipient, recvC} {
				got := receivedToday(t, store, recipient)
				if got != tc.want[recipient] {
					t.Errorf("%s received %d, want %d", recipient, got, tc.want[recipient])
				}
				total += got
			}
			if got := givenToday(t, store, testGiver); got != total {
				t.Errorf("giver gave %d, want %d", got, total)
			}
			if got := len(fake.posted()); got != len(tc.want) {
				t.Errorf("posted %d confirmations, want %d: %q", got, len(tc.want), fake.posted())
			}
		})
	}
}

func TestHandleMessageEventConfirmationTexts(t *testing.T) {
	ep, _, fake := newTestEventProcessor(t)
	ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
	ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer: :beer:"), "")

	want := []string{
		fmt.Sprintf("<@%s> gave 1 beer to <@%s>!", testGiver, testRecipient),
		fmt.Sprintf("<@%s> gave 2 beers to <@%s>!", testGiver, testRecipient),
	}
	got := fake.posted()
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("posted %q, want %q", got, want)
	}
}

func TestHandleMessageEventIgnores(t *testing.T) {
	text := mention(testRecipient) + " :beer:"
	cases := map[string]*slackevents.MessageEvent{
		"other channel": msgEvent("C0OTHER", testGiver, text),
		"no user":       msgEvent(testChannel, "", text),
		"bot message":   {Channel: testChannel, User: testGiver, Text: text, TimeStamp: uniqueTS(), SubType: "bot_message"},
		"edited":        {Channel: testChannel, User: testGiver, Text: text, TimeStamp: uniqueTS(), SubType: "message_changed"},
		"self gift":     msgEvent(testChannel, testGiver, mention(testGiver)+" :beer:"),
	}
	for name, ev := range cases {
		t.Run(name, func(t *testing.T) {
			ep, store, fake := newTestEventProcessor(t)
			ep.handleMessageEvent(ev, "")
			if got := givenToday(t, store, testGiver); got != 0 {
				t.Errorf("gave %d beers", got)
			}
			if got := fake.posted(); len(got) != 0 {
				t.Errorf("posted %q", got)
			}
		})
	}
}

func TestHandleMessageEventDailyLimit(t *testing.T) {
	t.Run("limit reached", func(t *testing.T) {
		ep, store, fake := newTestEventProcessor(t)
		ep.maxPerDay = 3
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer: :beer: :beer:"), "")
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")

		if got := givenToday(t, store, testGiver); got != 3 {
			t.Errorf("gave %d, want 3", got)
		}
		posted := fake.posted()
		if len(posted) != 2 || !strings.Contains(posted[1], "reached your daily limit of 3 beers") {
			t.Errorf("posted %q", posted)
		}
	})

	t.Run("would exceed", func(t *testing.T) {
		ep, store, fake := newTestEventProcessor(t)
		ep.maxPerDay = 3
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer: :beer: :beer:"), "")

		if got := givenToday(t, store, testGiver); got != 1 {
			t.Errorf("gave %d, want 1 (the second message is rejected as a whole)", got)
		}
		posted := fake.posted()
		if len(posted) != 2 || !strings.Contains(posted[1], "trying to give 3 beers, but you only have 2 left") {
			t.Errorf("posted %q", posted)
		}
	})

	t.Run("exactly the limit is allowed", func(t *testing.T) {
		ep, store, _ := newTestEventProcessor(t)
		ep.maxPerDay = 2
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer: :beer:"), "")
		if got := givenToday(t, store, testGiver); got != 2 {
			t.Errorf("gave %d, want 2", got)
		}
	})

	t.Run("limit is per giver", func(t *testing.T) {
		ep, store, _ := newTestEventProcessor(t)
		ep.maxPerDay = 1
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
		ep.handleMessageEvent(msgEvent(testChannel, "UOTHERGIV", mention(testRecipient)+" :beer:"), "")
		if got := receivedToday(t, store, testRecipient); got != 2 {
			t.Errorf("recipient got %d, want 2", got)
		}
	})
}

func TestHandleMessageEventDeduplication(t *testing.T) {
	t.Run("same envelope", func(t *testing.T) {
		ep, store, fake := newTestEventProcessor(t)
		ev := msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:")
		for i := 0; i < 3; i++ {
			ep.handleMessageEvent(ev, "env-1")
		}
		if got := givenToday(t, store, testGiver); got != 1 || len(fake.posted()) != 1 {
			t.Errorf("gave %d beers, posted %d messages, want 1 and 1", got, len(fake.posted()))
		}
	})

	t.Run("without envelope the channel, user and ts identify the event", func(t *testing.T) {
		ep, store, fake := newTestEventProcessor(t)
		ev := msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:")
		ep.handleMessageEvent(ev, "")
		ep.handleMessageEvent(ev, "")
		if got := givenToday(t, store, testGiver); got != 1 || len(fake.posted()) != 1 {
			t.Errorf("gave %d beers, posted %d messages, want 1 and 1", got, len(fake.posted()))
		}
		marked, err := store.IsEventProcessed(fmt.Sprintf("msg|%s|%s|%s", ev.Channel, ev.User, ev.TimeStamp))
		if err != nil || !marked {
			t.Errorf("fallback event id not stored: %v, %v", marked, err)
		}
	})

	t.Run("different envelopes are different events", func(t *testing.T) {
		ep, store, _ := newTestEventProcessor(t)
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "env-a")
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "env-b")
		if got := givenToday(t, store, testGiver); got != 2 {
			t.Errorf("gave %d, want 2", got)
		}
	})

	t.Run("event is marked before side effects", func(t *testing.T) {
		// A message without beers is still marked (the mark precedes parsing), so a redelivery
		// is dropped without any work.
		ep, store, _ := newTestEventProcessor(t)
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, "no beer here"), "env-plain")
		if marked, _ := store.IsEventProcessed("env-plain"); !marked {
			t.Errorf("event not marked as processed")
		}
	})

	t.Run("counter counts first deliveries only", func(t *testing.T) {
		ep, _, _ := newTestEventProcessor(t)
		ep.msgsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_msgs_total"}, []string{"channel"})
		ev := msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:")
		ep.handleMessageEvent(ev, "env-count")
		ep.handleMessageEvent(ev, "env-count")
		if got := counterValue(t, ep.msgsProcessed.WithLabelValues(testChannel)); got != 1 {
			t.Errorf("counter = %v, want 1", got)
		}
	})
}

func TestHandleEventIgnoresUnusableEvents(t *testing.T) {
	ep, store, fake := newTestEventProcessor(t)

	ep.HandleEvent(socketmode.Event{Type: socketmode.EventTypeConnecting})
	ep.HandleEvent(socketmode.Event{Type: socketmode.EventTypeHello})
	ep.HandleEvent(socketmode.Event{Type: socketmode.EventTypeEventsAPI}) // nil request
	ep.HandleEvent(socketmode.Event{
		Type:    socketmode.EventTypeEventsAPI,
		Data:    "not an events api event",
		Request: &socketmode.Request{EnvelopeID: "env-bad-type"},
	})

	// Without a socket client the event cannot be acked and is dropped.
	ev := socketmodeEvent(t, "env-no-socket", messageEvent(testChannel, testGiver, mention(testRecipient)+" :beer:", uniqueTS()))
	ep.slackManager.socketClient = nil
	ep.HandleEvent(ev)

	if got := givenToday(t, store, testGiver); got != 0 || len(fake.posted()) != 0 {
		t.Errorf("gave %d beers, posted %q", got, fake.posted())
	}
}

func TestWorkerPoolProcessesAllEvents(t *testing.T) {
	ep, store, fake := newTestEventProcessor(t)
	ep.StartWorkers(3)

	givers := []string{"UGIVERAAA", "UGIVERBBB", "UGIVERCCC", "UGIVERDDD", "UGIVEREEE"}
	for round := 0; round < 4; round++ {
		for _, g := range givers {
			ep.handleMessageEvent(msgEvent(testChannel, g, mention(testRecipient)+" :beer:"), "")
		}
	}
	shutdownOrFail(t, ep)

	if got := receivedToday(t, store, testRecipient); got != 20 {
		t.Errorf("recipient got %d, want 20", got)
	}
	if got := len(fake.posted()); got != 20 {
		t.Errorf("posted %d confirmations, want 20", got)
	}
}

// The daily limit check and the write of one giver must not interleave: the events of one
// giver run on one worker, in order. With 4 workers and 15 messages of one giver, exactly 10
// beers (the limit) are stored, never more.
func TestWorkerPoolKeepsDailyLimitPerGiver(t *testing.T) {
	ep, store, fake := newTestEventProcessor(t)
	ep.maxPerDay = 10
	ep.StartWorkers(4)

	for i := 0; i < 15; i++ {
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
	}
	shutdownOrFail(t, ep)

	if got := givenToday(t, store, testGiver); got != 10 {
		t.Errorf("gave %d, want exactly the limit of 10", got)
	}
	limitMessages := 0
	for _, m := range fake.posted() {
		if strings.Contains(m, "daily limit") {
			limitMessages++
		}
	}
	if limitMessages != 5 {
		t.Errorf("%d limit messages, want 5 (%q)", limitMessages, fake.posted())
	}
}

func TestWorkerPoolDeduplicatesBeforeDispatch(t *testing.T) {
	ep, store, fake := newTestEventProcessor(t)
	ep.StartWorkers(2)

	ev := msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:")
	for i := 0; i < 5; i++ {
		ep.handleMessageEvent(ev, "env-dup")
	}
	shutdownOrFail(t, ep)

	if got := givenToday(t, store, testGiver); got != 1 || len(fake.posted()) != 1 {
		t.Errorf("gave %d beers, posted %d messages, want 1 and 1", got, len(fake.posted()))
	}
}

func TestWorkerPoolRunsJobsOfDifferentGiversInParallel(t *testing.T) {
	ep, _, _ := newTestEventProcessor(t)
	ep.StartWorkers(8)

	// Find two givers that map to different shards, block the first one's worker and check that
	// the second still runs.
	shardOf := func(user string) int {
		h := fnvShard(user, 8)
		return h
	}
	a := "UPARALLEL0"
	b := ""
	for i := 1; i < 100; i++ {
		if c := fmt.Sprintf("UPARALLEL%d", i); shardOf(c) != shardOf(a) {
			b = c
			break
		}
	}
	if b == "" {
		t.Fatal("no giver on another shard found")
	}

	release := make(chan struct{})
	started := make(chan struct{})
	ep.dispatch(a, func() { close(started); <-release })
	<-started

	ran := make(chan struct{})
	ep.dispatch(b, func() { close(ran) })
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("job of another giver was blocked by a slow job")
	}
	close(release)
	shutdownOrFail(t, ep)
}

func TestWorkerPoolSameGiverRunsInOrder(t *testing.T) {
	ep, _, _ := newTestEventProcessor(t)
	ep.StartWorkers(4)

	var mu sync.Mutex
	var order []int
	for i := 0; i < 100; i++ {
		ep.dispatch(testGiver, func() {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
		})
	}
	shutdownOrFail(t, ep)

	for i, v := range order {
		if v != i {
			t.Fatalf("order[%d] = %d", i, v)
		}
	}
	if len(order) != 100 {
		t.Fatalf("ran %d jobs, want 100", len(order))
	}
}

func TestWorkerPoolRecoversFromPanic(t *testing.T) {
	ep, _, _ := newTestEventProcessor(t)
	ep.StartWorkers(1)

	ep.dispatch(testGiver, func() { panic("boom") })
	done := make(chan struct{})
	ep.dispatch(testGiver, func() { close(done) })
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker died after a panic")
	}
	shutdownOrFail(t, ep)
}

func TestShutdownDrainsQueueAndLaterEventsRunInline(t *testing.T) {
	ep, store, _ := newTestEventProcessor(t)
	ep.StartWorkers(1)

	release := make(chan struct{})
	ep.dispatch(testGiver, func() { <-release })
	for i := 0; i < 5; i++ {
		ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
	}

	// The queue is not drained while the first job blocks: Shutdown must report the timeout...
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := ep.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown with a stuck worker = %v, want deadline exceeded", err)
	}
	// ...and finish the queued work once the worker is free.
	close(release)
	shutdownOrFail(t, ep)
	if got := givenToday(t, store, testGiver); got != 5 {
		t.Errorf("gave %d, want 5 (queued events must not be lost)", got)
	}

	// After shutdown nothing is queued any more: new events are handled inline.
	ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
	if got := givenToday(t, store, testGiver); got != 6 {
		t.Errorf("gave %d after shutdown, want 6", got)
	}
	shutdownOrFail(t, ep) // idempotent
}

func TestStartWorkersEdgeCases(t *testing.T) {
	ep, store, _ := newTestEventProcessor(t)
	ep.StartWorkers(0)
	ep.StartWorkers(-2)
	if ep.shards != nil {
		t.Fatal("n < 1 must keep inline processing")
	}
	ep.handleMessageEvent(msgEvent(testChannel, testGiver, mention(testRecipient)+" :beer:"), "")
	if got := givenToday(t, store, testGiver); got != 1 {
		t.Errorf("inline processing gave %d, want 1 (synchronously)", got)
	}

	ep.StartWorkers(2)
	first := ep.shards
	ep.StartWorkers(5)
	if len(ep.shards) != 2 || &ep.shards[0] != &first[0] {
		t.Errorf("a second StartWorkers must be a no-op")
	}
	shutdownOrFail(t, ep)

	ep.StartWorkers(3) // after shutdown: no restart
	if len(ep.shards) != 2 {
		t.Errorf("StartWorkers after Shutdown changed the pool")
	}

	// Shutdown without a pool returns immediately.
	ep2, _, _ := newTestEventProcessor(t)
	shutdownOrFail(t, ep2)
}

func shutdownOrFail(t *testing.T, ep *EventProcessor) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ep.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

// fnvShard mirrors the shard assignment of dispatch.
func fnvShard(user string, n int) int {
	ep := &EventProcessor{shards: make([]chan func(), n)}
	return ep.shardIndex(user)
}

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("read counter: %v", err)
	}
	return m.GetCounter().GetValue()
}
