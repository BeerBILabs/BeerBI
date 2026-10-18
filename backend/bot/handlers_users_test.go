package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// newUserTestHandlers builds handlers with a real SQLite store, optional Redis
// (miniredis) and a fake Slack user API.
func newUserTestHandlers(t *testing.T, withRedis bool, sl *fakeSlackUsers) (*APIHandlers, *SQLiteStore) {
	t.Helper()
	store := newTestStore(t)
	h := &APIHandlers{store: store, logger: zerolog.Nop()}
	var rc userRedisCache
	if withRedis {
		h.redisCache, _ = newTestRedis(t)
		rc = h.redisCache
	}
	h.users = NewUserResolver(store, rc, sl, zerolog.Nop())
	t.Cleanup(h.users.bg.Wait)
	return h, store
}

func TestNewAPIHandlers_WiresResolverWithNilDependencies(t *testing.T) {
	h := NewAPIHandlers(nil, nil, nil, nil, zerolog.Nop())
	if h.users == nil {
		t.Fatal("resolver not created")
	}
	if h.users.store != nil || h.users.redis != nil || h.users.slack != nil {
		t.Error("nil pointers must become nil interfaces")
	}
	if got := h.users.Resolve(t.Context(), []string{"U1"}, time.Second); len(got) != 0 {
		t.Errorf("resolver without layers returned %v", got)
	}
}

func TestUserHandler(t *testing.T) {
	sl := newFakeSlackUsers(0)
	sl.add("UONE", "One User")
	h, store := newUserTestHandlers(t, true, sl)
	if err := store.SetCachedUser("UCACHED", "Cached User", "img-c"); err != nil {
		t.Fatal(err)
	}

	if rec := serve(h.UserHandler, "/api/user"); rec.Code != http.StatusBadRequest {
		t.Errorf("missing user: %d", rec.Code)
	}
	if rec := serve(h.UserHandler, "/api/user?user=UNOPE"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown user: %d", rec.Code)
	}

	rec := serve(h.UserHandler, "/api/user?user=UONE")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	got := decodeBody[map[string]string](t, rec)
	if got["real_name"] != "One User" || got["profile_image"] == "" {
		t.Errorf("body = %v", got)
	}

	got = decodeBody[map[string]string](t, serve(h.UserHandler, "/api/user?user=UCACHED"))
	if got["real_name"] != "Cached User" || got["profile_image"] != "img-c" {
		t.Errorf("cached body = %v", got)
	}
	if sl.callsFor("UCACHED") != 0 {
		t.Error("cached user fetched from Slack")
	}
}

func TestUserHandler_SlackTimeoutAndErrorAre503(t *testing.T) {
	sl := newFakeSlackUsers(300 * time.Millisecond)
	sl.add("USLOW", "Slow User")
	sl.fail["UERR"] = errors.New("internal_error")
	h, _ := newUserTestHandlers(t, false, sl)
	h.userWait = 20 * time.Millisecond

	rec := serve(h.UserHandler, "/api/user?user=USLOW")
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") != "5" {
		t.Errorf("slow Slack: status %d, Retry-After %q; want 503, 5", rec.Code, rec.Header().Get("Retry-After"))
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("503 Cache-Control = %q, want no-store", cc)
	}

	h.userWait = time.Second
	for range 2 {
		rec = serve(h.UserHandler, "/api/user?user=UERR")
		if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
			t.Errorf("Slack error: status %d, Retry-After %q; want 503 with Retry-After", rec.Code, rec.Header().Get("Retry-After"))
		}
	}
	if c := sl.callsFor("UERR"); c != 1 {
		t.Errorf("users.info calls for failing ID = %d, want 1", c)
	}

	// Once the slow lookup finished it is served from the cache.
	h.users.bg.Wait()
	if rec = serve(h.UserHandler, "/api/user?user=USLOW"); rec.Code != http.StatusOK {
		t.Errorf("after lookup finished: status %d", rec.Code)
	}
}

func TestUserHandler_UnknownUserIs404AndNegativeCached(t *testing.T) {
	sl := newFakeSlackUsers(0)
	h, _ := newUserTestHandlers(t, false, sl)
	for range 3 {
		rec := serve(h.UserHandler, "/api/user?user=UNOPE")
		if rec.Code != http.StatusNotFound || rec.Header().Get("Retry-After") != "" {
			t.Errorf("unknown user: status %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
		}
		h.users.bg.Wait()
	}
	if c := sl.callsFor("UNOPE"); c != 1 {
		t.Errorf("users.info calls = %d, want 1", c)
	}
}

func TestBatchUsersHandler_Validation(t *testing.T) {
	h, _ := newUserTestHandlers(t, false, newFakeSlackUsers(0))
	if rec := serve(h.BatchUsersHandler, "/api/users"); rec.Code != http.StatusBadRequest {
		t.Errorf("missing ids: %d", rec.Code)
	}
	if rec := serve(h.BatchUsersHandler, "/api/users?ids=,%20,"); rec.Code != http.StatusBadRequest {
		t.Errorf("blank ids: %d", rec.Code)
	}
}

func TestBatchUsersHandler_CapsAt100(t *testing.T) {
	sl := newFakeSlackUsers(0)
	ids := testIDs(120)
	for _, id := range ids {
		sl.add(id, "Name "+id)
	}
	h, _ := newUserTestHandlers(t, false, sl)
	got := decodeBody[map[string]UserInfo](t, serve(h.BatchUsersHandler, "/api/users?ids="+strings.Join(ids, ",")))
	if len(got) != 100 {
		t.Errorf("got %d users, want 100", len(got))
	}
	if _, ok := got[ids[100]]; ok {
		t.Error("ID beyond the cap was resolved")
	}
}

// Audit §5: /api/users with 100 cold IDs answers in < 2 s with 100 ms Slack latency.
func TestBatchUsersHandler_100ColdIDsUnder2s(t *testing.T) {
	sl := newFakeSlackUsers(100 * time.Millisecond)
	ids := testIDs(100)
	for _, id := range ids {
		sl.add(id, "Name "+id)
	}
	h, store := newUserTestHandlers(t, true, sl)

	start := time.Now()
	rec := serve(h.BatchUsersHandler, "/api/users?ids="+strings.Join(ids, ","))
	took := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	got := decodeBody[map[string]UserInfo](t, rec)
	if len(got) != 100 || got[ids[42]].RealName != "Name "+ids[42] {
		t.Fatalf("resolved %d of 100", len(got))
	}
	if took >= 2*time.Second {
		t.Errorf("took %v, want < 2s", took)
	}
	t.Logf("/api/users, 100 cold IDs, 100 ms Slack latency: %v", took)

	// Warm second request: served from Redis without Slack.
	start = time.Now()
	_ = serve(h.BatchUsersHandler, "/api/users?ids="+strings.Join(ids, ","))
	t.Logf("/api/users, 100 warm IDs: %v", time.Since(start))
	if n := sl.totalCalls(); n != 100 {
		t.Errorf("slack calls = %d, want 100", n)
	}
	h.users.bg.Wait()
	if cached, _ := store.GetCachedUsers(ids); len(cached) != 100 {
		t.Errorf("sqlite has %d of 100", len(cached))
	}
}

// seedNamedStats adds beers for today and caches names for some of the users.
func seedNamedStats(t *testing.T, store *SQLiteStore) {
	t.Helper()
	ts := mustTime(t, time.Now().Format("2006-01-02")+"T09:00:00Z")
	beers := []struct {
		giver, recipient string
		count            int
	}{{"UGA", "URA", 3}, {"UGB", "URA", 1}, {"UGA", "UNONAME", 2}}
	for i, b := range beers {
		if err := store.AddBeer(b.giver, b.recipient, fmt.Sprintf("1700000000.%06d", i), ts, b.count); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetCachedUsers(map[string]UserInfo{
		"UGA": {RealName: "Giver A", ProfileImage: "img-ga"},
		"UGB": {RealName: "Giver B", ProfileImage: "img-gb"},
		"URA": {RealName: "Recipient A", ProfileImage: "img-ra"},
	}); err != nil {
		t.Fatal(err)
	}
}

func todayRange() string {
	d := time.Now().Format("2006-01-02")
	return "start=" + d + "&end=" + d
}

type namedTopJSON struct {
	UserID       string  `json:"userId"`
	Count        int     `json:"count"`
	RealName     *string `json:"real_name"`
	ProfileImage *string `json:"profile_image"`
}

type namedPairJSON struct {
	Giver          string  `json:"giver"`
	Recipient      string  `json:"recipient"`
	Count          int     `json:"count"`
	GiverName      string  `json:"giver_name"`
	GiverImage     string  `json:"giver_image"`
	RecipientName  *string `json:"recipient_name"`
	RecipientImage *string `json:"recipient_image"`
}

func TestTopUsersHandler_InlineNames(t *testing.T) {
	for _, withRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%v", withRedis), func(t *testing.T) {
			h, store := newUserTestHandlers(t, withRedis, newFakeSlackUsers(0))
			seedNamedStats(t, store)
			if withRedis {
				// Serve the leaderboard from the Redis "today" range.
				if err := h.redisCache.PopulateFromDB(t.Context(), store); err != nil {
					t.Fatal(err)
				}
			}

			got := decodeBody[struct {
				Givers     []namedTopJSON `json:"givers"`
				Recipients []namedTopJSON `json:"recipients"`
			}](t, serve(h.TopUsersHandler, "/api/stats/top?"+todayRange()))

			if len(got.Givers) != 2 || got.Givers[0].UserID != "UGA" || got.Givers[0].Count != 5 {
				t.Fatalf("givers = %+v", got.Givers)
			}
			if got.Givers[0].RealName == nil || *got.Givers[0].RealName != "Giver A" || *got.Givers[0].ProfileImage != "img-ga" {
				t.Errorf("giver name missing: %+v", got.Givers[0])
			}
			var unnamed *namedTopJSON
			for i := range got.Recipients {
				if got.Recipients[i].UserID == "UNONAME" {
					unnamed = &got.Recipients[i]
				}
			}
			if unnamed == nil || unnamed.RealName != nil || unnamed.ProfileImage != nil {
				t.Errorf("unresolvable user must omit name fields: %+v", unnamed)
			}
		})
	}
}

// The ETag covers the inline names, and a revalidation (304) does not ask
// Slack again for a user it could not resolve.
func TestTopUsersHandler_ETagCoversNamesAndSparesSlack(t *testing.T) {
	sl := newFakeSlackUsers(0)
	h, store := newUserTestHandlers(t, false, sl)
	seedNamedStats(t, store)
	target := "/api/stats/top?" + todayRange()

	first := serve(h.TopUsersHandler, target)
	wantStatus(t, first, http.StatusOK)
	etag := first.Header().Get("ETag")
	if etag == "" || !strings.Contains(first.Body.String(), `"real_name":"Giver A"`) {
		t.Fatalf("ETag %q, body %s", etag, first.Body.String())
	}
	h.users.bg.Wait()

	for range 3 {
		wantStatus(t, serve(h.TopUsersHandler, target, "If-None-Match", etag), http.StatusNotModified)
	}
	if c := sl.callsFor("UNONAME"); c != 1 {
		t.Errorf("users.info calls for the unresolvable user = %d, want 1", c)
	}

	// A name that appears later changes the body and therefore the ETag.
	if err := store.SetCachedUser("UNONAME", "Named Later", "img-n"); err != nil {
		t.Fatal(err)
	}
	if rec := serve(h.TopUsersHandler, target, "If-None-Match", etag); rec.Code != http.StatusOK || rec.Header().Get("ETag") == etag {
		t.Errorf("new name: status %d, ETag unchanged %v", rec.Code, rec.Header().Get("ETag") == etag)
	}
}

func TestPairsHandler_InlineNames(t *testing.T) {
	h, store := newUserTestHandlers(t, false, newFakeSlackUsers(0))
	seedNamedStats(t, store)

	got := decodeBody[[]namedPairJSON](t, serve(h.PairsHandler, "/api/stats/pairs?"+todayRange()))
	if len(got) != 3 {
		t.Fatalf("pairs = %+v", got)
	}
	for _, p := range got {
		if p.GiverName == "" || p.GiverImage == "" {
			t.Errorf("giver name missing: %+v", p)
		}
		if p.Recipient == "UNONAME" && (p.RecipientName != nil || p.RecipientImage != nil) {
			t.Errorf("unresolvable recipient must omit name fields: %+v", p)
		}
		if p.Recipient == "URA" && (p.RecipientName == nil || *p.RecipientName != "Recipient A") {
			t.Errorf("recipient name missing: %+v", p)
		}
	}
}

func TestCombinedAnalyticsHandler_InlineNames(t *testing.T) {
	sl := newFakeSlackUsers(0)
	sl.add("UNONAME", "Fetched From Slack") // resolved within statsNamesWait
	h, store := newUserTestHandlers(t, false, sl)
	seedNamedStats(t, store)

	rec := serve(h.CombinedAnalyticsHandler, "/api/stats/combined?"+todayRange())
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeBody[struct {
		Timeline      []map[string]any `json:"timeline"`
		Heatmap       []map[string]any `json:"heatmap"`
		TopGivers     []namedTopJSON   `json:"top_givers"`
		TopRecipients []namedTopJSON   `json:"top_recipients"`
		Pairs         []namedPairJSON  `json:"pairs"`
	}](t, rec)

	if len(got.Timeline) == 0 || len(got.Heatmap) == 0 {
		t.Errorf("non-user fields lost: timeline %d, heatmap %d", len(got.Timeline), len(got.Heatmap))
	}
	if len(got.TopGivers) != 2 || got.TopGivers[1].RealName == nil || *got.TopGivers[1].RealName != "Giver B" {
		t.Errorf("top givers = %+v", got.TopGivers)
	}
	for _, r := range got.TopRecipients {
		if r.RealName == nil || *r.RealName == "" {
			t.Errorf("recipient without name: %+v", r)
		}
	}
	if len(got.Pairs) != 3 {
		t.Fatalf("pairs = %+v", got.Pairs)
	}
	for _, p := range got.Pairs {
		if p.RecipientName == nil || *p.RecipientName == "" {
			t.Errorf("pair recipient without name: %+v", p)
		}
	}
	if sl.callsFor("UNONAME") != 1 || sl.totalCalls() != 1 {
		t.Errorf("slack calls = %+v, want only the uncached user once", sl.calls)
	}
}

func TestWithUserNames_NilAndEmpty(t *testing.T) {
	h := &APIHandlers{logger: zerolog.Nop()} // no resolver: names are simply omitted
	top := h.withUserNamesTop(t.Context(), nil)
	if top.Givers != nil || top.Recipients != nil {
		t.Errorf("nil result = %+v", top)
	}
	b, err := json.Marshal(top)
	if err != nil || string(b) != `{"givers":null,"recipients":null}` {
		t.Errorf("nil JSON = %s, %v", b, err)
	}
	if p := h.withUserNamesPairs(t.Context(), nil); p != nil {
		t.Errorf("nil pairs = %v", p)
	}
	named := h.withUserNamesTop(t.Context(), &TopUsersResult{Givers: []TopUserStats{{UserID: "U1", Count: 1}}})
	b, _ = json.Marshal(named.Givers)
	if string(b) != `[{"userId":"U1","count":1}]` {
		t.Errorf("unresolved entry JSON = %s", b)
	}
}
