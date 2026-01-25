package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

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

func FromChallenge(c *entity.Challenge) ChallengeResponse {
	return ChallengeResponse{
		Id:          c.Id.String(),
		Title:       c.Title,
		Description: c.Description,
		Category:    c.Category,
		Points:      c.Points,
		SolveCount:  c.SolveCount,
		IsHidden:    c.IsHidden,
	}
}

// FromChallengeWithSolved creates ChallengeResponse from ChallengeWithSolved
func FromChallengeWithSolved(cws *repo.ChallengeWithSolved) ChallengeResponse {
	res := FromChallenge(cws.Challenge)
	res.Solved = cws.Solved
	return res
}

// FromChallengeList creates list of ChallengeResponse
func FromChallengeList(items []*repo.ChallengeWithSolved) []ChallengeResponse {
	res := make([]ChallengeResponse, len(items))
	for i, item := range items {
		res[i] = FromChallengeWithSolved(item)
	}
	return res
}

type ScoreboardEntryResponse struct {
	TeamId     string  `json:"team_id"`
	TeamName   string  `json:"team_name"`
	Points     int     `json:"points"`
	LastSolved *string `json:"last_solved,omitempty"`
}

// FromScoreboardEntry creates ScoreboardEntryResponse from entity
func FromScoreboardEntry(e *repo.ScoreboardEntry) ScoreboardEntryResponse {
	res := ScoreboardEntryResponse{
		TeamId:   e.TeamId.String(),
		TeamName: e.TeamName,
		Points:   e.Points,
	}
	if !e.SolvedAt.IsZero() {
		ts := e.SolvedAt.Format("2006-01-02T15:04:05Z07:00")
		res.LastSolved = &ts
	}
	return res
}

// FromScoreboardList creates list of ScoreboardEntryResponse
func FromScoreboardList(items []*repo.ScoreboardEntry) []ScoreboardEntryResponse {
	res := make([]ScoreboardEntryResponse, len(items))
	for i, item := range items {
		res[i] = FromScoreboardEntry(item)
	}
	return res
}

type FirstBloodResponse struct {
	UserId   string `json:"user_id"`
	Username string `json:"username"`
	TeamId   string `json:"team_id"`
	TeamName string `json:"team_name"`
	SolvedAt string `json:"solved_at"`
}

// FromFirstBlood creates FirstBloodResponse from entity
func FromFirstBlood(fb *repo.FirstBloodEntry) FirstBloodResponse {
	return FirstBloodResponse{
		UserId:   fb.UserId.String(),
		Username: fb.Username,
		TeamId:   fb.TeamId.String(),
		TeamName: fb.TeamName,
		SolvedAt: fb.SolvedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
