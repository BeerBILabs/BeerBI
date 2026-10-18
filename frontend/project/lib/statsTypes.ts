// Response shapes of the backend stats endpoints as the frontend uses them.
//
// Inline user names: once the backend joins its user cache into the stats responses (audit B6.4),
// each entry may carry the display name (and avatar URL) directly. The fields are optional so the
// frontend works with both the old and the new backend; when a name is missing it falls back to
// resolving the ID through /api/proxy/users (lib/userNames.ts).

/** One row of /api/stats/top (and top_givers/top_recipients of /api/stats/combined). */
export type TopUserStats = {
  userId: string
  count: number
  /** Display name (Slack real name), when the backend provides it inline. */
  name?: string | null
  /** Avatar URL, when the backend provides it inline; null means "no avatar". */
  avatar?: string | null
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
  /** Display names, when the backend provides them inline. */
  giverName?: string | null
  recipientName?: string | null
}

/** Response of the frontend route /api/leaderboard (app/api/leaderboard/route.ts). */
export type LeaderboardResponse = {
  current: TopUsersResult
  /** Ranking of the comparison period, or null if none was requested or it failed. */
  previous: TopUsersResult | null
}
