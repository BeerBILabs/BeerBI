package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// testClock is a settable clock for the resolver (safe for its background goroutines).
type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func newTestClock() *testClock { return &testClock{t: time.Now()} }

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// ageCachedUser sets the updated_at of a user_cache row to age ago.
func ageCachedUser(t *testing.T, s *SQLiteStore, id string, age time.Duration) {
	t.Helper()
	old := time.Now().Add(-age).UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE user_cache SET updated_at = ? WHERE user_id = ?`, old, id); err != nil {
		t.Fatal(err)
	}
}

func cachedUpdatedAt(t *testing.T, s *SQLiteStore, id string) time.Time {
	t.Helper()
	got, err := s.GetCachedUsers([]string{id})
	if err != nil {
		t.Fatal(err)
	}
	return got[id].UpdatedAt
}

// resolveTimes runs Resolve n times and waits for the background lookups after each.
func resolveTimes(t *testing.T, r *UserResolver, id string, n int) {
	t.Helper()
	for range n {
		if got := r.Resolve(context.Background(), []string{id}, time.Second); got[id].RealName != "Former User" {
			t.Fatalf("cached name not served: %+v", got)
		}
		r.bg.Wait()
	}
}

func TestResolve_StaleRowWithoutSlackNameRefreshedOnce(t *testing.T) {
	for _, tc := range []struct {
		name      string
		withRedis bool
		slackSays func(*fakeSlackUsers)
	}{
		{"empty name, sqlite only", false, func(f *fakeSlackUsers) { f.add("UGONE", "") }},
		{"user_not_found, sqlite only", false, func(*fakeSlackUsers) {}},
		{"empty name, with redis", true, func(f *fakeSlackUsers) { f.add("UGONE", "") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			var rc userRedisCache
			var redisCache *RedisUserCache
			if tc.withRedis {
				redisCache, _ = newTestRedis(t)
				rc = redisCache
			}
			sl := newFakeSlackUsers(0)
			tc.slackSays(sl)
			if err := store.SetCachedUser("UGONE", "Former User", "img"); err != nil {
				t.Fatal(err)
			}
			ageCachedUser(t, store, "UGONE", 8*24*time.Hour)
			clock := newTestClock()
			r := NewUserResolver(store, rc, sl, zerolog.Nop())
			r.now = clock.now

			resolveTimes(t, r, "UGONE", 3)
			if c := sl.callsFor("UGONE"); c != 1 {
				t.Fatalf("users.info calls = %d, want 1", c)
			}
			// The cached name was rewritten, so the row is fresh again ...
			if age := time.Since(cachedUpdatedAt(t, store, "UGONE")); age > time.Hour {
				t.Errorf("updated_at not refreshed, age %v", age)
			}
			// ... and stays quiet after the negative window as well.
			clock.advance(negativeCacheTTL + time.Minute)
			resolveTimes(t, r, "UGONE", 2)
			if c := sl.callsFor("UGONE"); c != 1 {
				t.Errorf("users.info calls after negative window = %d, want 1", c)
			}
			if redisCache != nil {
				if u, _ := redisCache.GetUser(context.Background(), "UGONE"); u == nil || u.RealName != "Former User" {
					t.Errorf("stale hit not written back to redis: %+v", u)
				}
			}
		})
	}
}

func TestResolve_StaleRowSlackErrorRetriedAfterFailureWindow(t *testing.T) {
	store := newTestStore(t)
	sl := newFakeSlackUsers(0)
	sl.fail["UGONE"] = errors.New("internal_error")
	if err := store.SetCachedUser("UGONE", "Former User", "img"); err != nil {
		t.Fatal(err)
	}
	ageCachedUser(t, store, "UGONE", 8*24*time.Hour)
	clock := newTestClock()
	r := NewUserResolver(store, nil, sl, zerolog.Nop())
	r.now = clock.now

	resolveTimes(t, r, "UGONE", 3)
	if c := sl.callsFor("UGONE"); c != 1 {
		t.Fatalf("users.info calls = %d, want 1 within the failure window", c)
	}
	// A transient error is no answer: the row stays stale and is retried later.
	if age := time.Since(cachedUpdatedAt(t, store, "UGONE")); age < 7*24*time.Hour {
		t.Errorf("updated_at refreshed after a Slack error, age %v", age)
	}
	clock.advance(failureCacheTTL + time.Second)
	resolveTimes(t, r, "UGONE", 1)
	if c := sl.callsFor("UGONE"); c != 2 {
		t.Errorf("users.info calls after failure window = %d, want 2", c)
	}
}

func TestResolveOne_UnknownIDNegativeCached(t *testing.T) {
	sl := newFakeSlackUsers(0)
	clock := newTestClock()
	r := NewUserResolver(newTestStore(t), nil, sl, zerolog.Nop())
	r.now = clock.now

	for range 3 {
		if _, err := r.ResolveOne(context.Background(), "UNOPE", time.Second); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("err = %v, want ErrUserNotFound", err)
		}
		r.bg.Wait()
	}
	if c := sl.callsFor("UNOPE"); c != 1 {
		t.Fatalf("users.info calls = %d, want 1 within the negative window", c)
	}
	if got := r.Resolve(context.Background(), []string{"UNOPE"}, time.Second); len(got) != 0 {
		t.Errorf("batch returned %v", got)
	}

	// The user joins later: after the window it is looked up again.
	sl.add("UNOPE", "Late Joiner")
	clock.advance(negativeCacheTTL + time.Second)
	info, err := r.ResolveOne(context.Background(), "UNOPE", time.Second)
	if err != nil || info.RealName != "Late Joiner" {
		t.Fatalf("after window: %+v, %v", info, err)
	}
	if c := sl.callsFor("UNOPE"); c != 2 {
		t.Errorf("users.info calls = %d, want 2", c)
	}
	r.bg.Wait()
	r.mu.Lock()
	e := r.recent["UNOPE"]
	r.mu.Unlock()
	if e.status != fetchFound || e.info.RealName != "Late Joiner" {
		t.Errorf("remembered result = %+v, want the found user", e)
	}
}

func TestResolveOne_TimeoutAndFailureAreUnavailable(t *testing.T) {
	sl := newFakeSlackUsers(200 * time.Millisecond)
	sl.add("USLOW", "Slow User")
	sl.fail["UERR"] = errors.New("internal_error")
	r := NewUserResolver(nil, nil, sl, zerolog.Nop())

	_, err := r.ResolveOne(context.Background(), "USLOW", 10*time.Millisecond)
	var un *UserLookupUnavailableError
	if !errors.As(err, &un) || un.RetryAfter != lookupRetryAfter {
		t.Fatalf("slow lookup: err = %v, want unavailable with %v", err, lookupRetryAfter)
	}
	r.bg.Wait()

	_, err = r.ResolveOne(context.Background(), "UERR", time.Second)
	if !errors.As(err, &un) || un.RetryAfter != failureCacheTTL {
		t.Fatalf("failed lookup: err = %v, want unavailable with %v", err, failureCacheTTL)
	}
	// Within the failure window the error is answered from memory.
	_, err = r.ResolveOne(context.Background(), "UERR", time.Second)
	if !errors.As(err, &un) || un.RetryAfter <= 0 || un.RetryAfter > failureCacheTTL {
		t.Fatalf("cached failure: err = %v", err)
	}
	if c := sl.callsFor("UERR"); c != 1 {
		t.Errorf("users.info calls = %d, want 1", c)
	}
	if un.Error() == "" {
		t.Error("empty error text")
	}
}

func TestResolve_RateLimitBeyondDeadlineAbortsImmediately(t *testing.T) {
	sl := newFakeSlackUsers(0)
	sl.add("ULIMIT", "Limited User")
	sl.rateLimitOnce["ULIMIT"] = time.Hour // longer than slackFetchTimeout
	r := NewUserResolver(nil, nil, sl, zerolog.Nop())

	start := time.Now()
	_, err := r.ResolveOne(context.Background(), "ULIMIT", 5*time.Second)
	r.bg.Wait()
	if took := time.Since(start); took > time.Second {
		t.Fatalf("lookup slept on a Retry-After beyond its deadline: %v", took)
	}
	var un *UserLookupUnavailableError
	if !errors.As(err, &un) {
		t.Errorf("err = %v, want unavailable", err)
	}
	if c := sl.callsFor("ULIMIT"); c != 1 {
		t.Errorf("users.info calls = %d, want 1 (no retry)", c)
	}
}

func TestWarm_NamelessMembersNeedNoUsersInfo(t *testing.T) {
	sl := newFakeSlackUsers(0)
	sl.members = []slack.User{testSlackUser("UW1", "Warm One"), testSlackUser("UBOT", "")}
	sl.add("UBOT", "")
	r := NewUserResolver(newTestStore(t), nil, sl, zerolog.Nop())
	if _, err := r.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}

	info, err := r.ResolveOne(context.Background(), "UBOT", time.Second)
	if err != nil || info.RealName != "" || info.ProfileImage == "" {
		t.Errorf("nameless member = %+v, %v", info, err)
	}
	if got := r.Resolve(context.Background(), []string{"UW1", "UBOT"}, time.Second); len(got) != 1 {
		t.Errorf("batch = %v, want only the named member", got)
	}
	if n := sl.totalCalls(); n != 0 {
		t.Errorf("users.info calls after warmup = %d, want 0", n)
	}
}

func TestRecentResults_SweepsExpiredEntries(t *testing.T) {
	clock := newTestClock()
	r := NewUserResolver(nil, nil, nil, zerolog.Nop())
	r.now = clock.now
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range testIDs(recentSoftLimit) {
		r.remember(id, recentResult{until: clock.now().Add(-time.Second)})
	}
	r.remember("UFRESH", recentResult{until: clock.now().Add(time.Minute)})
	if len(r.recent) != 1 {
		t.Errorf("remembered results = %d after sweep, want 1", len(r.recent))
	}
}
