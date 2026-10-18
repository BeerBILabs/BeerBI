package main

import "context"

// Inline user names for the stats responses (audit B6.4).
//
// The stats handlers call one of the withUserNames* helpers right before
// encoding. Each entry gets the display data of its user(s) from the
// UserResolver cascade (Redis -> SQLite -> Slack, Slack waited for at most
// statsNamesWait). Name fields are omitted when a user cannot be resolved;
// clients then fall back to /api/users for those IDs.

// NamedTopUser is a TopUserStats entry with the user's display data:
//
//	{"userId":"U…","count":12,"real_name":"…","profile_image":"https://…"}
type NamedTopUser struct {
	TopUserStats
	RealName     string `json:"real_name,omitempty"`
	ProfileImage string `json:"profile_image,omitempty"`
}

// NamedPair is a PairStats entry with the display data of both users:
//
//	{"giver":"U…","recipient":"U…","count":3,
//	 "giver_name":"…","giver_image":"https://…",
//	 "recipient_name":"…","recipient_image":"https://…"}
type NamedPair struct {
	PairStats
	GiverName      string `json:"giver_name,omitempty"`
	GiverImage     string `json:"giver_image,omitempty"`
	RecipientName  string `json:"recipient_name,omitempty"`
	RecipientImage string `json:"recipient_image,omitempty"`
}

// NamedTopUsersResult is the /api/stats/top response with inline names.
// The embedded result keeps any other field; the named lists shadow the plain ones.
type NamedTopUsersResult struct {
	TopUsersResult
	Givers     []NamedTopUser `json:"givers"`
	Recipients []NamedTopUser `json:"recipients"`
}

// NamedCombinedAnalyticsResponse is the /api/stats/combined response with inline names.
// The embedded response keeps every other field; the named lists shadow the plain ones.
type NamedCombinedAnalyticsResponse struct {
	CombinedAnalyticsResponse
	TopGivers     []NamedTopUser `json:"top_givers"`
	TopRecipients []NamedTopUser `json:"top_recipients"`
	Pairs         []NamedPair    `json:"pairs"`
}

// withUserNamesTop adds inline names to a /api/stats/top result.
func (h *APIHandlers) withUserNamesTop(ctx context.Context, data *TopUsersResult) NamedTopUsersResult {
	if data == nil {
		data = &TopUsersResult{}
	}
	names := h.statsUserNames(ctx, topUserIDs(data.Givers), topUserIDs(data.Recipients))
	return NamedTopUsersResult{
		TopUsersResult: *data,
		Givers:         nameTopUsers(data.Givers, names),
		Recipients:     nameTopUsers(data.Recipients, names),
	}
}

// withUserNamesPairs adds inline names to a /api/stats/pairs result.
func (h *APIHandlers) withUserNamesPairs(ctx context.Context, pairs []PairStats) []NamedPair {
	return namePairs(pairs, h.statsUserNames(ctx, pairUserIDs(pairs)))
}

// withUserNamesCombined adds inline names to a /api/stats/combined response.
func (h *APIHandlers) withUserNamesCombined(ctx context.Context, resp CombinedAnalyticsResponse) NamedCombinedAnalyticsResponse {
	names := h.statsUserNames(ctx, topUserIDs(resp.TopGivers), topUserIDs(resp.TopRecipients), pairUserIDs(resp.Pairs))
	return NamedCombinedAnalyticsResponse{
		CombinedAnalyticsResponse: resp,
		TopGivers:                 nameTopUsers(resp.TopGivers, names),
		TopRecipients:             nameTopUsers(resp.TopRecipients, names),
		Pairs:                     namePairs(resp.Pairs, names),
	}
}

// statsUserNames resolves all IDs in one resolver call.
func (h *APIHandlers) statsUserNames(ctx context.Context, idLists ...[]string) map[string]UserInfo {
	var ids []string
	for _, l := range idLists {
		ids = append(ids, l...)
	}
	if len(ids) == 0 {
		return map[string]UserInfo{}
	}
	return h.users.Resolve(ctx, ids, statsNamesWait)
}

func topUserIDs(list []TopUserStats) []string {
	ids := make([]string, len(list))
	for i, u := range list {
		ids[i] = u.UserID
	}
	return ids
}

func pairUserIDs(list []PairStats) []string {
	ids := make([]string, 0, 2*len(list))
	for _, p := range list {
		ids = append(ids, p.Giver, p.Recipient)
	}
	return ids
}

// nameTopUsers keeps nil as nil so the JSON shape ("null" vs "[]") does not change.
func nameTopUsers(list []TopUserStats, names map[string]UserInfo) []NamedTopUser {
	if list == nil {
		return nil
	}
	out := make([]NamedTopUser, len(list))
	for i, u := range list {
		n := names[u.UserID]
		out[i] = NamedTopUser{TopUserStats: u, RealName: n.RealName, ProfileImage: n.ProfileImage}
	}
	return out
}

func namePairs(list []PairStats, names map[string]UserInfo) []NamedPair {
	if list == nil {
		return nil
	}
	out := make([]NamedPair, len(list))
	for i, p := range list {
		g, r := names[p.Giver], names[p.Recipient]
		out[i] = NamedPair{
			PairStats:      p,
			GiverName:      g.RealName,
			GiverImage:     g.ProfileImage,
			RecipientName:  r.RealName,
			RecipientImage: r.ProfileImage,
		}
	}
	return out
}
