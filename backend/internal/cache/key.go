package cache

import "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"

const (
	// KeyScoreboard is the Redis key for the live overall scoreboard cache.
	KeyScoreboard = cacheutil.KeyScoreboard
	// KeyScoreboardFrozen is the Redis key for the frozen (snapshot) scoreboard cache.
	KeyScoreboardFrozen = cacheutil.KeyScoreboardFrozen
	// KeyScoreboardFrozenPrefix is the prefix for time-stamped frozen scoreboard cache keys.
	KeyScoreboardFrozenPrefix = cacheutil.KeyScoreboardFrozenPrefix
	// KeyScoreboardBracketPrefix is the prefix for per-bracket live scoreboard cache keys.
	KeyScoreboardBracketPrefix = cacheutil.KeyScoreboardBracketPrefix
	// KeyCompetition is the Redis key for the cached competition state.
	KeyCompetition = cacheutil.KeyCompetition
	// KeyCompetitionGuard is the Redis key used as a distributed lock/guard for competition transitions.
	KeyCompetitionGuard = cacheutil.KeyCompetitionGuard
	// KeyAppSettings is the Redis key for the cached application settings.
	KeyAppSettings = cacheutil.KeyAppSettings
	// KeyFailedLoginPrefix is the prefix for per-identifier failed-login attempt counters.
	KeyFailedLoginPrefix = cacheutil.KeyFailedLoginPrefix
	// KeyLimiterPrefix is the prefix for rate-limiter token-bucket keys.
	KeyLimiterPrefix = cacheutil.KeyLimiterPrefix
	// PubSubScoreboard is the Redis pub/sub channel name for scoreboard update notifications.
	PubSubScoreboard = cacheutil.PubSubScoreboard
	// KeyAvatarUserPrefix is the prefix for user avatar cache keys.
	KeyAvatarUserPrefix = cacheutil.KeyAvatarUserPrefix
	// KeyAvatarTeamPrefix is the prefix for team avatar cache keys.
	KeyAvatarTeamPrefix = cacheutil.KeyAvatarTeamPrefix
)

// KeyUser returns the Redis cache key for a user by their string ID.
func KeyUser(userID string) string {
	return cacheutil.KeyUser(userID)
}

// KeyTeam returns the Redis cache key for a team by their string ID.
func KeyTeam(teamID string) string {
	return cacheutil.KeyTeam(teamID)
}

// KeyScoreboardBracket returns the Redis cache key for the live scoreboard of a specific bracket.
func KeyScoreboardBracket(bracketID string) string {
	return cacheutil.KeyScoreboardBracket(bracketID)
}

// KeyScoreboardBracketFrozen returns the Redis cache key for the frozen scoreboard of a specific bracket.
func KeyScoreboardBracketFrozen(bracketID string) string {
	return cacheutil.KeyScoreboardBracketFrozen(bracketID)
}

// KeyScoreboardFrozenAt returns the Redis cache key for a frozen scoreboard snapshot taken at the given Unix timestamp.
func KeyScoreboardFrozenAt(freezeUnix int64) string {
	return cacheutil.KeyScoreboardFrozenAt(freezeUnix)
}

// KeyScoreboardBracketFrozenAt returns the Redis cache key for a bracket's frozen scoreboard snapshot taken at the given Unix timestamp.
func KeyScoreboardBracketFrozenAt(bracketID string, freezeUnix int64) string {
	return cacheutil.KeyScoreboardBracketFrozenAt(bracketID, freezeUnix)
}

// KeyAvatarUser returns the Redis cache key for a user's avatar by their string ID.
func KeyAvatarUser(userID string) string {
	return cacheutil.KeyAvatarUser(userID)
}

// KeyAvatarTeam returns the Redis cache key for a team's avatar by their string ID.
func KeyAvatarTeam(teamID string) string {
	return cacheutil.KeyAvatarTeam(teamID)
}
