package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromSubmission(s *domain.SubmissionWithDetails) openapi.SubmissionResponse {
	res := openapi.SubmissionResponse{
		ID:                new(s.ID.String()),
		UserID:            new(s.UserID.String()),
		ChallengeID:       new(s.ChallengeID.String()),
		SubmittedFlag:     new(s.SubmittedFlag),
		IsCorrect:         new(s.IsCorrect),
		CreatedAt:         new(s.CreatedAt.Format(time.RFC3339)),
		Username:          new(s.Username),
		TeamName:          new(s.TeamName),
		ChallengeTitle:    new(s.ChallengeTitle),
		ChallengeCategory: new(s.ChallengeCategory),
	}
	if s.TeamID != nil {
		res.TeamID = new(s.TeamID.String())
	}

	if s.IP != "" {
		res.IP = new(s.IP)
	}

	return res
}

func FromSubmissionPublic(s *domain.SubmissionWithDetails) openapi.SubmissionResponse {
	res := openapi.SubmissionResponse{
		ID:                new(s.ID.String()),
		UserID:            new(s.UserID.String()),
		ChallengeID:       new(s.ChallengeID.String()),
		IsCorrect:         new(s.IsCorrect),
		CreatedAt:         new(s.CreatedAt.Format(time.RFC3339)),
		Username:          new(s.Username),
		TeamName:          new(s.TeamName),
		ChallengeTitle:    new(s.ChallengeTitle),
		ChallengeCategory: new(s.ChallengeCategory),
	}
	if s.TeamID != nil {
		res.TeamID = new(s.TeamID.String())
	}

	return res
}

func FromSubmissionListPublic(items []*domain.SubmissionWithDetails, total int64, page, perPage int) openapi.SubmissionListResponse {
	data, meta := BuildListResponse(items, FromSubmissionPublic, total, page, perPage)

	return openapi.SubmissionListResponse{Data: &data, Meta: meta}
}

func FromSubmissionList(items []*domain.SubmissionWithDetails, total int64, page, perPage int) openapi.SubmissionListResponse {
	data, meta := BuildListResponse(items, FromSubmission, total, page, perPage)

	return openapi.SubmissionListResponse{Data: &data, Meta: meta}
}

func FromSubmissionStats(stats *domain.SubmissionStats) openapi.SubmissionStatsResponse {
	return openapi.SubmissionStatsResponse{
		Total:     new(stats.Total),
		Correct:   new(stats.Correct),
		Incorrect: new(stats.Incorrect),
	}
}
