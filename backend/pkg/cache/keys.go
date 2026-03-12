package cache

import "strconv"

const (
	KeyScoreboard              = "scoreboard"
	KeyScoreboardFrozen        = "scoreboard:frozen"
	KeyScoreboardFrozenPrefix  = "scoreboard:frozen:"
	KeyScoreboardBracketPrefix = "scoreboard:bracket:"
	KeyCompetition             = "competition"
	KeyAppSettings             = "app_settings"
	KeyFailedLoginPrefix       = "failed_login:"
	KeyLimiterPrefix           = "limiter:"
	PubSubScoreboard           = "scoreboard:updates"
)

func KeyUser(userID string) string {
	return "user:" + userID
}

func KeyTeam(teamID string) string {
	return "team:" + teamID
}

func KeyScoreboardBracket(bracketID string) string {
	return "scoreboard:bracket:" + bracketID
}

func KeyScoreboardBracketFrozen(bracketID string) string {
	return "scoreboard:frozen:bracket:" + bracketID
}

func KeyScoreboardFrozenAt(freezeUnix int64) string {
	return KeyScoreboardFrozenPrefix + strconv.FormatInt(freezeUnix, 10)
}

func KeyScoreboardBracketFrozenAt(bracketID string, freezeUnix int64) string {
	return KeyScoreboardFrozenPrefix + strconv.FormatInt(freezeUnix, 10) + ":bracket:" + bracketID
}
