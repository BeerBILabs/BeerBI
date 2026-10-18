// Response shapes of the backend stats endpoints as the frontend uses them.
//
// Inline user names (audit B6.4): the backend joins its user cache into the stats responses
// (backend/bot/stats_names.go). Each entry then carries the display name and avatar URL of its
// user(s). The fields use `omitempty` on the backend:
//   - name field missing: the backend could not resolve the user; the frontend looks the ID up
//     through /api/proxy/users (lib/userNames.ts).
//   - name present, image field missing: the user has no avatar.

/** One row of /api/stats/top (and top_givers/top_recipients of /api/stats/combined). */
export type TopUserStats = {
  userId: string
  count: number
  /** Display name (Slack real name), when the backend resolved the user. */
  real_name?: string | null
  /** Avatar URL; missing next to a name means "no avatar". */
  profile_image?: string | null
}

export type TopUsersResult = {
  givers: TopUserStats[]
  recipients: TopUserStats[]
}

/** One row of /api/stats/pairs (and pairs of /api/stats/combined). */
export type PairStats = {
  giver: string
  recipient: string
  count: number
  /** Display data of both users, when the backend resolved them (same rules as TopUserStats). */
  giver_name?: string | null
  giver_image?: string | null
  recipient_name?: string | null
  recipient_image?: string | null
}

/** Response of the frontend route /api/leaderboard (app/api/leaderboard/route.ts). */
export type LeaderboardResponse = {
  current: TopUsersResult
  /** Ranking of the comparison period, or null if none was requested or it failed. */
  previous: TopUsersResult | null
}
