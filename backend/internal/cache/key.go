package cache

import "strconv"

const (
	// KeyScoreboard is the Redis key for the live overall scoreboard cache.
	KeyScoreboard = "scoreboard"
	// KeyScoreboardFrozen is the Redis key for the frozen (snapshot) scoreboard cache.
	KeyScoreboardFrozen = "scoreboard:frozen"
	// KeyScoreboardFrozenPrefix is the prefix for time-stamped frozen scoreboard cache keys.
	KeyScoreboardFrozenPrefix = "scoreboard:frozen:"
	// KeyScoreboardBracketPrefix is the prefix for per-bracket live scoreboard cache keys.
	KeyScoreboardBracketPrefix = "scoreboard:bracket:"
	// KeyCompetition is the Redis key for the cached competition state.
	KeyCompetition = "competition"
	// KeyCompetitionGuard is the Redis key used as a distributed lock/guard for competition transitions.
	KeyCompetitionGuard = "competition:guard"
	// KeyAppSettings is the Redis key for the cached application settings.
	KeyAppSettings = "app_settings"
	// KeyFailedLoginPrefix is the prefix for per-identifier failed-login attempt counters.
	KeyFailedLoginPrefix = "failed_login:"
	// KeyLimiterPrefix is the prefix for rate-limiter token-bucket keys.
	KeyLimiterPrefix = "limiter:"
	// PubSubScoreboard is the Redis pub/sub channel name for scoreboard update notifications.
	PubSubScoreboard = "scoreboard:updates"
	// KeyAvatarUserPrefix is the prefix for user avatar cache keys.
	KeyAvatarUserPrefix = "avatar:user:"
	// KeyAvatarTeamPrefix is the prefix for team avatar cache keys.
	KeyAvatarTeamPrefix = "avatar:team:"
)

// KeyUser returns the Redis cache key for a user by their string ID.
func KeyUser(userID string) string {
	return "user:" + userID
}

// KeyTeam returns the Redis cache key for a team by their string ID.
func KeyTeam(teamID string) string {
	return "team:" + teamID
}

// KeyScoreboardBracket returns the Redis cache key for the live scoreboard of a specific bracket.
func KeyScoreboardBracket(bracketID string) string {
	return "scoreboard:bracket:" + bracketID
}

// KeyScoreboardBracketFrozen returns the Redis cache key for the frozen scoreboard of a specific bracket.
func KeyScoreboardBracketFrozen(bracketID string) string {
	return "scoreboard:frozen:bracket:" + bracketID
}

// KeyScoreboardFrozenAt returns the Redis cache key for a frozen scoreboard snapshot taken at the given Unix timestamp.
func KeyScoreboardFrozenAt(freezeUnix int64) string {
	return KeyScoreboardFrozenPrefix + strconv.FormatInt(freezeUnix, 10)
}

// KeyScoreboardBracketFrozenAt returns the Redis cache key for a bracket's frozen scoreboard snapshot taken at the given Unix timestamp.
func KeyScoreboardBracketFrozenAt(bracketID string, freezeUnix int64) string {
	return KeyScoreboardFrozenPrefix + strconv.FormatInt(freezeUnix, 10) + ":bracket:" + bracketID
}

// KeyAvatarUser returns the Redis cache key for a user's avatar by their string ID.
func KeyAvatarUser(userID string) string {
	return KeyAvatarUserPrefix + userID
}

// KeyAvatarTeam returns the Redis cache key for a team's avatar by their string ID.
func KeyAvatarTeam(teamID string) string {
	return KeyAvatarTeamPrefix + teamID
}
