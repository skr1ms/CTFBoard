package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase"
)

func FromChallenge(c *entity.Challenge) openapi.ResponseChallengeResponse {
	return openapi.ResponseChallengeResponse{
		ID:          ptr(c.ID.String()),
		Title:       ptr(c.Title),
		Description: ptr(c.Description),
		Category:    ptr(c.Category),
		Points:      ptr(c.Points),
		SolveCount:  ptr(c.SolveCount),
		IsHidden:    ptr(c.IsHidden),
	}
}

func FromChallengeWithSolved(cws *repo.ChallengeWithSolved) openapi.ResponseChallengeResponse {
	res := FromChallenge(cws.Challenge)
	res.Solved = ptr(cws.Solved)
	return res
}

func FromChallengeWithTags(cwt *usecase.ChallengeWithTags) openapi.ResponseChallengeResponse {
	res := FromChallengeWithSolved(cwt.ChallengeWithSolved)
	if len(cwt.Tags) > 0 {
		tags := make([]openapi.ResponseTagResponse, len(cwt.Tags))
		for i, t := range cwt.Tags {
			tags[i] = FromTag(t)
		}
		res.Tags = &tags
	}
	return res
}

func FromChallengeList(items []*usecase.ChallengeWithTags) []openapi.ResponseChallengeResponse {
	res := make([]openapi.ResponseChallengeResponse, len(items))
	for i, item := range items {
		res[i] = FromChallengeWithTags(item)
	}
	return res
}

func FromTag(t *entity.Tag) openapi.ResponseTagResponse {
	return openapi.ResponseTagResponse{
		ID:    ptr(t.ID.String()),
		Name:  ptr(t.Name),
		Color: ptr(t.Color),
	}
}

func FromTagList(items []*entity.Tag) []openapi.ResponseTagResponse {
	res := make([]openapi.ResponseTagResponse, len(items))
	for i, item := range items {
		res[i] = FromTag(item)
	}
	return res
}

// FromScoreboardEntry creates ScoreboardEntryResponse from entity
func FromScoreboardEntry(e *repo.ScoreboardEntry) openapi.ResponseScoreboardEntryResponse {
	res := openapi.ResponseScoreboardEntryResponse{
		TeamID:   ptr(e.TeamID.String()),
		TeamName: ptr(e.TeamName),
		Points:   ptr(e.Points),
	}
	if !e.SolvedAt.IsZero() {
		ts := e.SolvedAt.Format(time.RFC3339)
		res.LastSolved = &ts
	}
	return res
}

// FromScoreboardList creates list of ScoreboardEntryResponse
func FromScoreboardList(items []*repo.ScoreboardEntry) []openapi.ResponseScoreboardEntryResponse {
	res := make([]openapi.ResponseScoreboardEntryResponse, len(items))
	for i, item := range items {
		res[i] = FromScoreboardEntry(item)
	}
	return res
}

// FromFirstBlood creates FirstBloodResponse from entity
func FromFirstBlood(fb *repo.FirstBloodEntry) openapi.ResponseFirstBloodResponse {
	return openapi.ResponseFirstBloodResponse{
		UserID:   ptr(fb.UserID.String()),
		Username: ptr(fb.Username),
		TeamID:   ptr(fb.TeamID.String()),
		TeamName: ptr(fb.TeamName),
		SolvedAt: ptr(fb.SolvedAt.Format(time.RFC3339)),
	}
}
