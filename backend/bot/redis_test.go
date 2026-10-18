package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func newTestRedis(t *testing.T) (*RedisUserCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialerRetries: 1})
	rc := newRedisUserCacheWithClient(client, zerolog.Nop())
	t.Cleanup(func() { _ = rc.Close() })
	return rc, mr
}

func TestNewRedisUserCache(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := NewRedisUserCache(mr.Addr(), zerolog.Nop())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	_ = rc.Close()

	// A port nobody listens on fails the ping.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	if _, err := NewRedisUserCache(addr, zerolog.Nop()); err == nil {
		t.Fatal("expected ping error for closed port")
	}
}

func TestRedisUser_SetGetDelete(t *testing.T) {
	ctx := context.Background()
	rc, mr := newTestRedis(t)

	if u, err := rc.GetUser(ctx, "UR1"); u != nil || err != nil {
		t.Fatalf("miss = %+v, %v; want nil, nil", u, err)
	}
	if err := rc.SetUser(ctx, "UR1", "Redis One", "img"); err != nil {
		t.Fatal(err)
	}
	u, err := rc.GetUser(ctx, "UR1")
	if err != nil || u == nil || u.RealName != "Redis One" || u.ProfileImage != "img" || u.CachedAt.IsZero() {
		t.Fatalf("get = %+v, %v", u, err)
	}
	if ttl := mr.TTL("user:UR1"); ttl != 7*24*time.Hour {
		t.Errorf("ttl = %v, want 7 days", ttl)
	}
	if err := rc.DeleteUser(ctx, "UR1"); err != nil {
		t.Fatal(err)
	}
	if mr.Exists("user:UR1") {
		t.Error("user not deleted")
	}

	if err := mr.Set("user:UBAD", "{not json"); err != nil {
		t.Fatal(err)
	}
	if _, err := rc.GetUser(ctx, "UBAD"); err == nil {
		t.Error("expected unmarshal error")
	}
}

func TestRedisUsers_PipelineBatch(t *testing.T) {
	ctx := context.Background()
	rc, mr := newTestRedis(t)

	if got, err := rc.GetUsers(ctx, nil); err != nil || len(got) != 0 {
		t.Fatalf("empty GetUsers = %v, %v", got, err)
	}
	if err := rc.SetUsers(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := rc.SetUsers(ctx, map[string]UserCacheData{
		"UB1": {RealName: "Batch One"},
		"UB2": {RealName: "Batch Two", ProfileImage: "img"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := mr.Set("user:UBAD", "{not json"); err != nil {
		t.Fatal(err)
	}

	got, err := rc.GetUsers(ctx, []string{"UB1", "UB2", "UMISS", "UBAD"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["UB1"].RealName != "Batch One" || got["UB2"].ProfileImage != "img" {
		t.Errorf("GetUsers = %+v", got)
	}
}

func TestLeaderboard_MissUntilPopulated(t *testing.T) {
	ctx := context.Background()
	rc, _ := newTestRedis(t)

	// C2: after a Redis restart the range is not populated. Reads must be a miss
	// (nil), even when increments since the restart already created the sorted set.
	g, err := rc.GetTopGivers(ctx, RangeAllTime, 10)
	if err != nil || g != nil {
		t.Fatalf("unpopulated givers = %v, %v; want nil, nil", g, err)
	}
	if err := rc.IncrementGivenStats(ctx, "UINC", 2); err != nil {
		t.Fatal(err)
	}
	if err := rc.IncrementReceivedStats(ctx, "UINC2", 2); err != nil {
		t.Fatal(err)
	}
	if g, _ := rc.GetTopGivers(ctx, RangeAllTime, 10); g != nil {
		t.Fatalf("partial set served as hit: %v", g)
	}
	if r, _ := rc.GetTopRecipients(ctx, RangeAllTime, 10); r != nil {
		t.Fatalf("partial set served as hit: %v", r)
	}

	store := newTestStore(t)
	now := time.Now()
	ts := now.Format("2006-01-02") + "T10:00:00Z"
	for i, b := range []struct {
		giver, recipient string
		count            int
	}{{"UG1", "UR1", 3}, {"UG2", "UR1", 1}, {"UG1", "UR2", 2}} {
		if err := store.AddBeer(b.giver, b.recipient, fmt.Sprintf("1700000000.%06d", i), mustTime(t, ts), b.count); err != nil {
			t.Fatal(err)
		}
	}
	if err := rc.PopulateFromDB(ctx, store); err != nil {
		t.Fatal(err)
	}

	g, err = rc.GetTopGivers(ctx, RangeAllTime, 10)
	if err != nil || len(g) != 2 || g[0].UserID != "UG1" || g[0].Count != 5 {
		t.Fatalf("givers = %+v, %v", g, err)
	}
	r, err := rc.GetTopRecipients(ctx, RangeAllTime, 1)
	if err != nil || len(r) != 1 || r[0].UserID != "UR1" || r[0].Count != 4 {
		t.Fatalf("recipients (limit 1) = %+v, %v", r, err)
	}

	// The populated last-quarter range is empty: a hit with an empty, non-nil list.
	lq, err := rc.GetTopGivers(ctx, RangeLastQuarter, 10)
	if err != nil || lq == nil || len(lq) != 0 {
		t.Fatalf("empty populated range = %#v, %v; want empty non-nil", lq, err)
	}
}

func TestLeaderboard_RestartIsMiss(t *testing.T) {
	ctx := context.Background()
	rc, mr := newTestRedis(t)
	if err := rc.PopulateFromDB(ctx, newTestStore(t)); err != nil {
		t.Fatal(err)
	}
	if g, _ := rc.GetTopGivers(ctx, RangeToday, 5); g == nil {
		t.Fatal("populated range should be a hit")
	}
	mr.FlushAll() // Redis restarted without persistence
	if g, err := rc.GetTopGivers(ctx, RangeToday, 5); g != nil || err != nil {
		t.Fatalf("after flush = %v, %v; want miss", g, err)
	}
}

func TestCircuitBreaker_OpensOnRedisFailure(t *testing.T) {
	ctx := context.Background()
	rc, mr := newTestRedis(t)
	mr.Close()

	for i := range 3 {
		if _, err := rc.GetTopGivers(ctx, RangeToday, 5); err == nil {
			t.Fatalf("read %d: expected error with Redis down", i)
		}
	}
	if !rc.circuitBreaker.IsOpen() {
		t.Fatal("circuit breaker should be open after 3 failures")
	}
	if _, err := rc.GetTopRecipients(ctx, RangeToday, 5); err == nil || err.Error() != "circuit breaker open" {
		t.Errorf("open breaker read err = %v", err)
	}
	if err := rc.IncrementGivenStats(ctx, "U1", 1); err != nil {
		t.Errorf("open breaker must skip writes silently, got %v", err)
	}
	if err := rc.IncrementReceivedStats(ctx, "U1", 1); err != nil {
		t.Errorf("open breaker must skip writes silently, got %v", err)
	}
}

func TestCircuitBreaker_Unit(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Hour)
	cb.RecordFailure()
	cb.RecordSuccess()
	cb.RecordFailure()
	if cb.IsOpen() {
		t.Fatal("success should reset the failure count")
	}
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Fatal("breaker should open at threshold")
	}
	cb.Reset()
	if cb.IsOpen() {
		t.Fatal("Reset should close the breaker")
	}
}

func TestRedisWrites_FailWhenRedisDown(t *testing.T) {
	ctx := context.Background()
	rc, mr := newTestRedis(t)
	mr.Close()
	if err := rc.SetUser(ctx, "U1", "N", ""); err == nil {
		t.Error("SetUser: expected error")
	}
	if err := rc.SetUsers(ctx, map[string]UserCacheData{"U1": {RealName: "N"}}); err == nil {
		t.Error("SetUsers: expected error")
	}
	if err := rc.DeleteUser(ctx, "U1"); err == nil {
		t.Error("DeleteUser: expected error")
	}
	if _, err := rc.GetUser(ctx, "U1"); err == nil {
		t.Error("GetUser: expected error")
	}
	if got, err := rc.GetUsers(ctx, []string{"U1"}); err != nil || len(got) != 0 {
		t.Errorf("GetUsers must degrade to empty, got %v, %v", got, err)
	}
	if err := rc.IncrementGivenStats(ctx, "U1", 1); err == nil {
		t.Error("IncrementGivenStats: expected error")
	}
	if err := rc.IncrementReceivedStats(ctx, "U1", 1); err == nil {
		t.Error("IncrementReceivedStats: expected error")
	}
}

func TestCalculateActiveRanges_AlwaysHasCurrentRanges(t *testing.T) {
	rc, _ := newTestRedis(t)
	got := map[string]bool{}
	for _, r := range rc.calculateActiveRanges() {
		got[r] = true
	}
	for _, want := range []string{RangeAllTime, RangeToday, RangeCurrentQuarter, RangeCurrentMonth, RangeLast7Days} {
		if !got[want] {
			t.Errorf("missing range %s", want)
		}
	}
}

func TestStartSyncWorker_PopulatesAndStops(t *testing.T) {
	rc, mr := newTestRedis(t)
	store := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		rc.StartSyncWorker(ctx, store, 10*time.Millisecond)
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !mr.Exists(leaderboardReadyKey(RangeAllTime)) {
		if time.Now().After(deadline) {
			t.Fatal("sync worker did not populate")
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(30 * time.Millisecond) // let the ticker run at least once
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sync worker did not stop")
	}
}
