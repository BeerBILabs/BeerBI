package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// UserInfo is the display data of a Slack user, as served by /api/user and
// /api/users and inlined into the stats responses.
type UserInfo struct {
	RealName     string `json:"real_name"`
	ProfileImage string `json:"profile_image"`
}

// SlackUserAPI is the part of the Slack Web API the user resolver needs.
// *slack.Client implements it; tests use a fake.
type SlackUserAPI interface {
	GetUserInfoContext(ctx context.Context, user string, options ...slack.GetUserInfoOption) (*slack.User, error)
	GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error)
}

// userCacheStore is the persistent user cache (SQLite user_cache table).
type userCacheStore interface {
	GetCachedUsers(userIDs []string) (map[string]CachedUser, error)
	SetCachedUsers(users map[string]UserInfo) error
}

// userRedisCache is the shared hot cache (Redis user:<id> keys).
type userRedisCache interface {
	GetUsers(ctx context.Context, userIDs []string) (map[string]*UserCacheData, error)
	SetUsers(ctx context.Context, users map[string]UserCacheData) error
}

const (
	// defaultSlackConcurrency bounds parallel users.info calls across all requests.
	// users.info is Tier 4 (100+ per minute, bursts allowed); 10 keeps 100 cold
	// IDs at about 1 s with 100 ms Slack latency.
	defaultSlackConcurrency = 10
	// usersRequestWait is how long /api/user(s) waits for Slack before answering
	// with what it has. Unfinished lookups keep running and fill the caches.
	usersRequestWait = 8 * time.Second
	// statsNamesWait is how long a stats response waits for Slack for names that
	// are in neither cache. After the users.list warmup this almost never happens.
	statsNamesWait = 1 * time.Second
	// slackFetchTimeout caps one background users.info lookup, including the
	// wait for a free worker and rate-limit back-off.
	slackFetchTimeout = 30 * time.Second
	// userCacheStaleAfter: older SQLite rows are still served but refreshed in
	// the background.
	userCacheStaleAfter = 7 * 24 * time.Hour
	// slackRateLimitRetries is how often a rate-limited users.info call is retried.
	slackRateLimitRetries = 2
)

// fetchCall is one in-flight users.info lookup, shared by every caller that
// asks for the same ID while it runs.
type fetchCall struct {
	done  chan struct{}
	info  UserInfo
	found bool // Slack returned the user; RealName may be empty for deactivated accounts
}

// UserResolver resolves Slack user IDs to display data with the cascade
// Redis -> SQLite user_cache -> Slack users.info. Slack lookups run on a
// bounded worker pool, are de-duplicated while in flight and write through
// to both caches. RunWarmup preloads both caches from users.list.
type UserResolver struct {
	store  userCacheStore // may be nil
	redis  userRedisCache // may be nil
	slack  SlackUserAPI   // may be nil
	logger zerolog.Logger

	sem          chan struct{}
	fetchTimeout time.Duration
	staleAfter   time.Duration
	now          func() time.Time

	mu       sync.Mutex
	inflight map[string]*fetchCall
	bg       sync.WaitGroup // running Slack lookups (tests wait on it)
}

// NewUserResolver creates a resolver. store, redis and slackAPI may be nil;
// the corresponding layer is then skipped.
func NewUserResolver(store userCacheStore, redis userRedisCache, slackAPI SlackUserAPI, logger zerolog.Logger) *UserResolver {
	return &UserResolver{
		store:        store,
		redis:        redis,
		slack:        slackAPI,
		logger:       logger.With().Str("component", "user_resolver").Logger(),
		sem:          make(chan struct{}, defaultSlackConcurrency),
		fetchTimeout: slackFetchTimeout,
		staleAfter:   userCacheStaleAfter,
		now:          time.Now,
		inflight:     make(map[string]*fetchCall),
	}
}

// Resolve returns display data for every ID it can resolve within wait.
// IDs without a known non-empty name are absent from the result; their Slack
// lookups continue in the background and fill the caches for the next call.
func (r *UserResolver) Resolve(ctx context.Context, userIDs []string, wait time.Duration) map[string]UserInfo {
	out := make(map[string]UserInfo, len(userIDs))
	if r == nil {
		return out
	}
	for id, info := range r.lookup(ctx, uniqueIDs(userIDs), wait) {
		if info.RealName != "" {
			out[id] = info
		}
	}
	return out
}

// ResolveOne resolves a single user. ok is false when neither cache has the
// user and Slack did not return it in time. Unlike Resolve it also returns a
// user whose Slack name is empty (deactivated account without a cache entry).
func (r *UserResolver) ResolveOne(ctx context.Context, userID string, wait time.Duration) (UserInfo, bool) {
	if r == nil {
		return UserInfo{}, false
	}
	info, ok := r.lookup(ctx, []string{userID}, wait)[userID]
	return info, ok
}

func (r *UserResolver) lookup(ctx context.Context, ids []string, wait time.Duration) map[string]UserInfo {
	out := make(map[string]UserInfo, len(ids))
	missing := r.fromRedis(ctx, ids, out)
	missing = r.fromStore(ctx, missing, out)
	if len(missing) == 0 || r.slack == nil {
		return out
	}

	calls := make(map[string]*fetchCall, len(missing))
	for _, id := range missing {
		calls[id] = r.fetchAsync(id)
	}

	wctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	for id, c := range calls {
		select {
		case <-c.done:
			if c.found {
				out[id] = c.info
			}
		case <-wctx.Done():
			collectFinished(calls, out)
			r.logger.Warn().Int("missing", len(missing)).Int("resolved", len(out)).Dur("wait", wait).
				Msg("slack lookups still running, answering with partial result")
			return out
		}
	}
	return out
}

// collectFinished copies every completed lookup into out without blocking.
func collectFinished(calls map[string]*fetchCall, out map[string]UserInfo) {
	for id, c := range calls {
		select {
		case <-c.done:
			if c.found {
				out[id] = c.info
			}
		default:
		}
	}
}

func (r *UserResolver) fromRedis(ctx context.Context, ids []string, out map[string]UserInfo) []string {
	if r.redis == nil || len(ids) == 0 {
		return ids
	}
	cached, err := r.redis.GetUsers(ctx, ids)
	if err != nil {
		r.logger.Warn().Err(err).Msg("redis user lookup failed")
		return ids
	}
	missing := make([]string, 0, len(ids))
	for _, id := range ids {
		if c, ok := cached[id]; ok && c != nil && c.RealName != "" {
			out[id] = UserInfo{RealName: c.RealName, ProfileImage: c.ProfileImage}
			continue
		}
		missing = append(missing, id)
	}
	return missing
}

func (r *UserResolver) fromStore(ctx context.Context, ids []string, out map[string]UserInfo) []string {
	if r.store == nil || len(ids) == 0 {
		return ids
	}
	cached, err := r.store.GetCachedUsers(ids)
	if err != nil {
		r.logger.Warn().Err(err).Msg("sqlite user lookup failed")
		return ids
	}
	missing := make([]string, 0, len(ids))
	backfill := make(map[string]UserCacheData)
	now := r.now()
	for _, id := range ids {
		c, ok := cached[id]
		if !ok || c.RealName == "" {
			missing = append(missing, id)
			continue
		}
		out[id] = UserInfo{RealName: c.RealName, ProfileImage: c.ProfileImage}
		if now.Sub(c.UpdatedAt) > r.staleAfter && r.slack != nil {
			r.fetchAsync(id) // serve the stale row, refresh in the background
		} else {
			backfill[id] = UserCacheData{RealName: c.RealName, ProfileImage: c.ProfileImage}
		}
	}
	if r.redis != nil && len(backfill) > 0 {
		if err := r.redis.SetUsers(ctx, backfill); err != nil {
			r.logger.Warn().Err(err).Msg("redis backfill from sqlite failed")
		}
	}
	return missing
}

// fetchAsync starts a users.info lookup for id, or joins the one already running.
func (r *UserResolver) fetchAsync(id string) *fetchCall {
	r.mu.Lock()
	if c, ok := r.inflight[id]; ok {
		r.mu.Unlock()
		return c
	}
	c := &fetchCall{done: make(chan struct{})}
	r.inflight[id] = c
	r.bg.Add(1)
	r.mu.Unlock()

	go func() {
		defer r.bg.Done()
		// Detached from the HTTP request on purpose: a lookup that outlives
		// its request still fills the caches.
		ctx, cancel := context.WithTimeout(context.Background(), r.fetchTimeout)
		defer cancel()

		c.info, c.found = r.fetchFromSlack(ctx, id)
		close(c.done) // answer waiting requests before the cache writes

		if c.found && c.info.RealName != "" {
			// Deactivated users can come back without a name; never overwrite a cached name with that.
			r.writeThrough(ctx, map[string]UserInfo{id: c.info})
		}
		// Leave the finished call visible until the caches hold the result,
		// so a request arriving in between does not start a second lookup.
		r.mu.Lock()
		delete(r.inflight, id)
		r.mu.Unlock()
	}()
	return c
}

// fetchFromSlack runs one users.info call on the worker pool. The worker slot
// is held only for the Slack call (and rate-limit back-off).
func (r *UserResolver) fetchFromSlack(ctx context.Context, id string) (UserInfo, bool) {
	select {
	case r.sem <- struct{}{}:
	case <-ctx.Done():
		r.logger.Warn().Str("userID", id).Msg("no free slack worker before timeout")
		return UserInfo{}, false
	}
	defer func() { <-r.sem }()

	var user *slack.User
	for attempt := 0; ; attempt++ {
		var err error
		user, err = r.slack.GetUserInfoContext(ctx, id, slack.GetUserInfoOptionIncludeLocale(false))
		if err == nil {
			break
		}
		var rl *slack.RateLimitedError
		if errors.As(err, &rl) && attempt < slackRateLimitRetries {
			r.logger.Warn().Str("userID", id).Dur("retry_after", rl.RetryAfter).Msg("slack rate limit, backing off")
			select {
			case <-time.After(rl.RetryAfter):
				continue
			case <-ctx.Done():
			}
		}
		r.logger.Warn().Str("userID", id).Err(err).Msg("slack users.info failed")
		return UserInfo{}, false
	}
	if user == nil {
		return UserInfo{}, false
	}
	return UserInfo{RealName: user.RealName, ProfileImage: user.Profile.Image192}, true
}

// writeThrough stores users in SQLite and Redis.
func (r *UserResolver) writeThrough(ctx context.Context, users map[string]UserInfo) {
	if len(users) == 0 {
		return
	}
	if r.store != nil {
		if err := r.store.SetCachedUsers(users); err != nil {
			r.logger.Warn().Err(err).Int("count", len(users)).Msg("failed to cache users to sqlite")
		}
	}
	if r.redis != nil {
		data := make(map[string]UserCacheData, len(users))
		for id, u := range users {
			data[id] = UserCacheData{RealName: u.RealName, ProfileImage: u.ProfileImage}
		}
		if err := r.redis.SetUsers(ctx, data); err != nil {
			r.logger.Warn().Err(err).Int("count", len(users)).Msg("failed to cache users to redis")
		}
	}
}

// Warm loads every workspace member via users.list (paginated, rate limits
// handled by slack-go) and writes them through to SQLite and Redis.
// Members without a real name are skipped so cached names of deactivated
// users survive.
func (r *UserResolver) Warm(ctx context.Context) (int, error) {
	if r == nil || r.slack == nil {
		return 0, nil
	}
	members, err := r.slack.GetUsersContext(ctx, slack.GetUsersOptionLimit(200))
	if err != nil {
		return 0, err
	}
	users := make(map[string]UserInfo, len(members))
	for _, m := range members {
		if m.ID == "" || m.RealName == "" {
			continue
		}
		users[m.ID] = UserInfo{RealName: m.RealName, ProfileImage: m.Profile.Image192}
	}
	r.writeThrough(ctx, users)
	return len(users), nil
}

// RunWarmup calls Warm now and then every interval until ctx is cancelled.
func (r *UserResolver) RunWarmup(ctx context.Context, interval time.Duration) {
	if r == nil || r.slack == nil {
		return
	}
	warm := func() {
		start := time.Now()
		n, err := r.Warm(ctx)
		if err != nil {
			r.logger.Warn().Err(err).Msg("users.list warmup failed")
			return
		}
		r.logger.Info().Int("users", n).Dur("took", time.Since(start)).Msg("user cache warmed from users.list")
	}
	warm()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			warm()
		case <-ctx.Done():
			return
		}
	}
}

// uniqueIDs drops empty and duplicate IDs, keeping the first occurrence order.
func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
