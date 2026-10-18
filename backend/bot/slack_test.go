package main

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestReconnectDelayBackoff(t *testing.T) {
	cases := map[int]time.Duration{
		0:   1 * time.Second,
		1:   2 * time.Second,
		2:   4 * time.Second,
		3:   8 * time.Second,
		8:   256 * time.Second,
		9:   maxReconnectDelay, // 512 s would exceed the cap
		10:  maxReconnectDelay,
		40:  maxReconnectDelay, // used to overflow the duration into a negative delay
		500: maxReconnectDelay,
	}
	for count, want := range cases {
		if got := reconnectDelay(count); got != want {
			t.Errorf("reconnectDelay(%d) = %v, want %v", count, got, want)
		}
	}
	if maxReconnectDelay != 5*time.Minute {
		t.Errorf("maxReconnectDelay = %v, want 5m", maxReconnectDelay)
	}
}

func TestSetConnectedResetsReconnectCount(t *testing.T) {
	scm := NewSlackConnectionManager("xoxb-test", "xapp-test", zerolog.Nop())
	scm.reconnectCount = 4

	scm.setConnected(false)
	if scm.IsConnected() || scm.reconnectCount != 4 {
		t.Errorf("disconnect: connected=%v count=%d, want false and 4 (count only resets once connected)", scm.IsConnected(), scm.reconnectCount)
	}
	scm.setConnected(true)
	if !scm.IsConnected() || scm.reconnectCount != 0 {
		t.Errorf("connect: connected=%v count=%d, want true and 0", scm.IsConnected(), scm.reconnectCount)
	}
}
