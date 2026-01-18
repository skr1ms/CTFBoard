package response

type ChallengeResponse struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Points      int    `json:"points"`
	SolveCount  int    `json:"solve_count"`
	IsHidden    bool   `json:"is_hidden"`
	Solved      bool   `json:"solved"`
}

type ScoreboardEntryResponse struct {
	TeamId     string  `json:"team_id"`
	TeamName   string  `json:"team_name"`
	Points     int     `json:"points"`
	LastSolved *string `json:"last_solved,omitempty"`
}

type FirstBloodResponse struct {
	UserId   string `json:"user_id"`
	Username string `json:"username"`
	TeamId   string `json:"team_id"`
	TeamName string `json:"team_name"`
	SolvedAt string `json:"solved_at"`
}
