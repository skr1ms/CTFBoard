package cache

const (
	KeyScoreboard       = "scoreboard"
	KeyScoreboardFrozen = "scoreboard:frozen"
	KeyCompetition      = "competition"
	KeyAppSettings      = "app_settings"
)

func KeyScoreboardBracket(bracketID string) string {
	return "scoreboard:bracket:" + bracketID
}

func KeyScoreboardBracketFrozen(bracketID string) string {
	return "scoreboard:frozen:bracket:" + bracketID
}
