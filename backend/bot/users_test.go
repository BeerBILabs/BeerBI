package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// fakeSlackUsers is an in-memory SlackUserAPI with configurable latency that
// records call counts and the peak number of concurrent users.info calls.
type fakeSlackUsers struct {
	mu            sync.Mutex
	users         map[string]slack.User
	latency       time.Duration
	rateLimitOnce map[string]bool
	calls         map[string]int
	active        int
	maxActive     int
	members       []slack.User
	listErr       error
}

func newFakeSlackUsers(latency time.Duration) *fakeSlackUsers {
	return &fakeSlackUsers{
		users:         map[string]slack.User{},
		latency:       latency,
		rateLimitOnce: map[string]bool{},
		calls:         map[string]int{},
	}
}

func testSlackUser(id, name string) slack.User {
	return slack.User{ID: id, RealName: name, Profile: slack.UserProfile{Image192: "https://example.invalid/" + id + ".png"}}
}

func (f *fakeSlackUsers) add(id, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[id] = testSlackUser(id, name)
}

func (f *fakeSlackUsers) GetUserInfoContext(ctx context.Context, user string, _ ...slack.GetUserInfoOption) (*slack.User, error) {
	f.mu.Lock()
	f.calls[user]++
	f.active++
	f.maxActive = max(f.maxActive, f.active)
	limited := f.rateLimitOnce[user]
	delete(f.rateLimitOnce, user)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		f.mu.Unlock()
	}()

	if limited {
		return nil, &slack.RateLimitedError{RetryAfter: 10 * time.Millisecond}
	}
	select {
	case <-time.After(f.latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[user]
	if !ok {
		return nil, errors.New("user_not_found")
	}
	return &u, nil
}

func (f *fakeSlackUsers) GetUsersContext(_ context.Context, _ ...slack.GetUsersOption) ([]slack.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.members, f.listErr
}

func (f *fakeSlackUsers) totalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		n += c
	}
	return n
}

func (f *fakeSlackUsers) callsFor(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[id]
}

func (f *fakeSlackUsers) peak() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxActive
}

// countingStore records which IDs were looked up in SQLite.
type countingStore struct {
	*SQLiteStore
	mu      sync.Mutex
	lookups [][]string
}

func (c *countingStore) GetCachedUsers(ids []string) (map[string]CachedUser, error) {
	c.mu.Lock()
	c.lookups = append(c.lookups, append([]string(nil), ids...))
	c.mu.Unlock()
	return c.SQLiteStore.GetCachedUsers(ids)
}

func testIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("UTEST%04d", i)
	}
	return ids
}

func TestResolve_CacheOrderRedisSQLiteSlack(t *testing.T) {
	ctx := context.Background()
	store := &countingStore{SQLiteStore: newTestStore(t)}
	rc, _ := newTestRedis(t)
	sl := newFakeSlackUsers(0)

	if err := rc.SetUser(ctx, "UREDIS", "Redis User", "img-r"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCachedUser("USQL", "Sql User", "img-s"); err != nil {
		t.Fatal(err)
	}
	sl.add("USLACK", "Slack User")
	// Slack also knows the cached users; it must not be asked for them.
	sl.add("UREDIS", "Wrong")
	sl.add("USQL", "Wrong")

	r := NewUserResolver(store, rc, sl, zerolog.Nop())
	got := r.Resolve(ctx, []string{"UREDIS", "USQL", "USLACK", "UNONE", "USQL", ""}, time.Second)

	want := map[string]string{"UREDIS": "Redis User", "USQL": "Sql User", "USLACK": "Slack User"}
	if len(got) != len(want) {
		t.Fatalf("got %d users, want %d: %+v", len(got), len(want), got)
	}
	for id, name := range want {
		if got[id].RealName != name {
			t.Errorf("%s: got %q, want %q", id, got[id].RealName, name)
		}
	}
	if sl.callsFor("UREDIS") != 0 || sl.callsFor("USQL") != 0 {
		t.Errorf("slack asked for cached users: %+v", sl.calls)
	}
	if sl.callsFor("USLACK") != 1 || sl.callsFor("UNONE") != 1 {
		t.Errorf("slack calls: %+v", sl.calls)
	}
	if len(store.lookups) != 1 || len(store.lookups[0]) != 3 {
		t.Errorf("sqlite lookups = %v, want one lookup without the redis hit", store.lookups)
	}

	// SQLite hit was backfilled into Redis, Slack hit written to both
	// (after the response, so wait for the background writes).
	r.bg.Wait()
	if u, _ := rc.GetUser(ctx, "USQL"); u == nil || u.RealName != "Sql User" {
		t.Errorf("redis backfill missing: %+v", u)
	}
	if u, _ := rc.GetUser(ctx, "USLACK"); u == nil || u.RealName != "Slack User" {
		t.Errorf("redis write-through missing: %+v", u)
	}
	if u, _ := store.GetCachedUser("USLACK"); u == nil || u.RealName != "Slack User" {
		t.Errorf("sqlite write-through missing: %+v", u)
	}
}

func TestResolve_StaleRowServedAndRefreshed(t *testing.T) {
	store := newTestStore(t)
	sl := newFakeSlackUsers(0)
	sl.add("USTALE", "New Name")
	if err := store.SetCachedUser("USTALE", "Old Name", "img"); err != nil {
		t.Fatal(err)
	}
	r := NewUserResolver(store, nil, sl, zerolog.Nop())
	r.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }

	got := r.Resolve(context.Background(), []string{"USTALE"}, time.Second)
	if got["USTALE"].RealName != "Old Name" {
		t.Fatalf("stale row not served: %+v", got)
	}
	r.bg.Wait()
	if u, _ := store.GetCachedUser("USTALE"); u == nil || u.RealName != "New Name" {
		t.Fatalf("stale row not refreshed: %+v", u)
	}
}

func TestResolve_DeactivatedUserKeepsCachedName(t *testing.T) {
	store := newTestStore(t)
	sl := newFakeSlackUsers(0)
	sl.add("UGONE", "") // deactivated: Slack answers without a name
	sl.add("UNEW", "")
	if err := store.SetCachedUser("UGONE", "Former User", "img"); err != nil {
		t.Fatal(err)
	}
	r := NewUserResolver(store, nil, sl, zerolog.Nop())
	r.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }

	got := r.Resolve(context.Background(), []string{"UGONE", "UNEW"}, time.Second)
	r.bg.Wait()
	if got["UGONE"].RealName != "Former User" {
		t.Errorf("cached name lost: %+v", got)
	}
	if _, ok := got["UNEW"]; ok {
		t.Errorf("batch result must omit users without a name: %+v", got)
	}
	if u, _ := store.GetCachedUser("UGONE"); u == nil || u.RealName != "Former User" {
		t.Errorf("empty Slack name overwrote cache: %+v", u)
	}

	info, ok := r.ResolveOne(context.Background(), "UNEW", time.Second)
	if !ok || info.RealName != "" || info.ProfileImage == "" {
		t.Errorf("ResolveOne = %+v, %v; want Slack data with empty name", info, ok)
	}
}

func TestResolve_100ColdIDsBoundedAndFast(t *testing.T) {
	sl := newFakeSlackUsers(100 * time.Millisecond)
	ids := testIDs(100)
	for _, id := range ids {
		sl.add(id, "Name "+id)
	}
	r := NewUserResolver(newTestStore(t), nil, sl, zerolog.Nop())

	start := time.Now()
	got := r.Resolve(context.Background(), ids, usersRequestWait)
	took := time.Since(start)

	if len(got) != 100 {
		t.Fatalf("resolved %d of 100", len(got))
	}
	if took >= 2*time.Second {
		t.Errorf("100 cold IDs took %v, want < 2s", took)
	}
	if p := sl.peak(); p > defaultSlackConcurrency {
		t.Errorf("peak concurrency %d exceeds pool size %d", p, defaultSlackConcurrency)
	}
	if n := sl.totalCalls(); n != 100 {
		t.Errorf("slack calls = %d, want 100", n)
	}
	t.Logf("100 cold IDs with 100 ms Slack latency: %v, peak concurrency %d", took, sl.peak())
}

func TestResolve_DeduplicatesConcurrentRequests(t *testing.T) {
	sl := newFakeSlackUsers(50 * time.Millisecond)
	ids := testIDs(10)
	for _, id := range ids {
		sl.add(id, "Name "+id)
	}
	r := NewUserResolver(nil, nil, sl, zerolog.Nop())

	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := r.Resolve(context.Background(), ids, time.Second); len(got) != 10 {
				t.Errorf("resolved %d of 10", len(got))
			}
		}()
	}
	wg.Wait()
	for _, id := range ids {
		if c := sl.callsFor(id); c != 1 {
			t.Errorf("%s fetched %d times, want 1", id, c)
		}
	}
}

func TestResolve_PartialResultAfterWaitThenCached(t *testing.T) {
	store := newTestStore(t)
	rc, _ := newTestRedis(t)
	sl := newFakeSlackUsers(300 * time.Millisecond)
	ids := testIDs(5)
	for _, id := range ids {
		sl.add(id, "Name "+id)
	}
	r := NewUserResolver(store, rc, sl, zerolog.Nop())

	start := time.Now()
	got := r.Resolve(context.Background(), ids, 20*time.Millisecond)
	if time.Since(start) > 200*time.Millisecond {
		t.Errorf("Resolve did not return after its wait budget")
	}
	if len(got) != 0 {
		t.Errorf("expected no results before Slack answers, got %d", len(got))
	}

	r.bg.Wait() // lookups outlive the request and fill the caches
	got = r.Resolve(context.Background(), ids, time.Second)
	if len(got) != 5 {
		t.Fatalf("second call resolved %d of 5", len(got))
	}
	if n := sl.totalCalls(); n != 5 {
		t.Errorf("slack calls = %d, want 5 (second call served from cache)", n)
	}
}

func TestResolve_RequestContextCancelled(t *testing.T) {
	sl := newFakeSlackUsers(200 * time.Millisecond)
	sl.add("UCTX", "Ctx User")
	r := NewUserResolver(nil, nil, sl, zerolog.Nop())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := r.Resolve(ctx, []string{"UCTX"}, time.Second); len(got) != 0 {
		t.Errorf("cancelled request returned %v", got)
	}
	r.bg.Wait()
}

func TestResolve_RetriesRateLimit(t *testing.T) {
	sl := newFakeSlackUsers(0)
	sl.add("URL", "Limited User")
	sl.rateLimitOnce["URL"] = true
	r := NewUserResolver(nil, nil, sl, zerolog.Nop())
	got := r.Resolve(context.Background(), []string{"URL"}, time.Second)
	if got["URL"].RealName != "Limited User" {
		t.Fatalf("rate-limited lookup not retried: %+v", got)
	}
	if c := sl.callsFor("URL"); c != 2 {
		t.Errorf("calls = %d, want 2", c)
	}
}

func TestResolve_FetchTimeoutWithoutFreeWorker(t *testing.T) {
	sl := newFakeSlackUsers(0)
	sl.add("UBUSY", "Busy User")
	r := NewUserResolver(nil, nil, sl, zerolog.Nop())
	r.sem = make(chan struct{}, 1)
	r.sem <- struct{}{} // pool exhausted
	r.fetchTimeout = 10 * time.Millisecond
	if got := r.Resolve(context.Background(), []string{"UBUSY"}, time.Second); len(got) != 0 {
		t.Errorf("got %v without a free worker", got)
	}
	if sl.totalCalls() != 0 {
		t.Errorf("slack called without a worker slot")
	}
}

func TestResolve_NilAndMissingLayers(t *testing.T) {
	var nilResolver *UserResolver
	if got := nilResolver.Resolve(context.Background(), []string{"U1"}, time.Second); len(got) != 0 {
		t.Errorf("nil resolver returned %v", got)
	}
	if _, ok := nilResolver.ResolveOne(context.Background(), "U1", time.Second); ok {
		t.Error("nil resolver ResolveOne ok")
	}
	if n, err := nilResolver.Warm(context.Background()); n != 0 || err != nil {
		t.Errorf("nil Warm = %d, %v", n, err)
	}
	nilResolver.RunWarmup(context.Background(), time.Hour) // must return immediately

	store := newTestStore(t)
	if err := store.SetCachedUser("UONLY", "Cached Only", ""); err != nil {
		t.Fatal(err)
	}
	r := NewUserResolver(store, nil, nil, zerolog.Nop())
	got := r.Resolve(context.Background(), []string{"UONLY", "UMISSING"}, time.Second)
	if len(got) != 1 || got["UONLY"].RealName != "Cached Only" {
		t.Errorf("cache-only resolver = %v", got)
	}
}

func TestWarm_WritesThroughAndSkipsNamelessMembers(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	rc, _ := newTestRedis(t)
	sl := newFakeSlackUsers(0)
	sl.members = []slack.User{
		testSlackUser("UW1", "Warm One"),
		testSlackUser("UW2", "Warm Two"),
		testSlackUser("UW3", ""),
		testSlackUser("", "No ID"),
	}
	if err := store.SetCachedUser("UW3", "Kept Name", ""); err != nil {
		t.Fatal(err)
	}
	r := NewUserResolver(store, rc, sl, zerolog.Nop())

	n, err := r.Warm(ctx)
	if err != nil || n != 2 {
		t.Fatalf("Warm = %d, %v; want 2, nil", n, err)
	}
	for _, id := range []string{"UW1", "UW2"} {
		if u, _ := store.GetCachedUser(id); u == nil || u.RealName == "" {
			t.Errorf("%s not in sqlite", id)
		}
		if u, _ := rc.GetUser(ctx, id); u == nil || u.RealName == "" {
			t.Errorf("%s not in redis", id)
		}
	}
	if u, _ := store.GetCachedUser("UW3"); u == nil || u.RealName != "Kept Name" {
		t.Errorf("nameless member overwrote cache: %+v", u)
	}

	// After the warmup a batch request needs no users.info call.
	if got := r.Resolve(ctx, []string{"UW1", "UW2"}, time.Second); len(got) != 2 || sl.totalCalls() != 0 {
		t.Errorf("after warmup: %d users, %d slack calls", len(got), sl.totalCalls())
	}

	sl.listErr = errors.New("missing_scope")
	if _, err := r.Warm(ctx); err == nil {
		t.Error("Warm swallowed users.list error")
	}
}

func TestRunWarmup_RunsImmediatelyAndStops(t *testing.T) {
	store := newTestStore(t)
	sl := newFakeSlackUsers(0)
	sl.members = []slack.User{testSlackUser("URW", "Run Warm")}
	r := NewUserResolver(store, nil, sl, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.RunWarmup(ctx, 10*time.Millisecond)
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if u, _ := store.GetCachedUser("URW"); u != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("warmup did not run")
		}
		time.Sleep(5 * time.Millisecond)
	}
	sl.mu.Lock()
	sl.listErr = errors.New("ratelimited") // ticker runs log and continue
	sl.mu.Unlock()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunWarmup did not stop on cancel")
	}
}

func TestUniqueIDs(t *testing.T) {
	got := uniqueIDs([]string{"B", "", "A", "B", "C", "A"})
	want := []string{"B", "A", "C"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("uniqueIDs = %v, want %v", got, want)
	}
}
