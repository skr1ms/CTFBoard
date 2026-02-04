package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromSubmission(s *entity.SubmissionWithDetails) openapi.ResponseSubmissionResponse {
	res := openapi.ResponseSubmissionResponse{
		ID:                ptr(s.ID.String()),
		UserID:            ptr(s.UserID.String()),
		ChallengeID:       ptr(s.ChallengeID.String()),
		SubmittedFlag:     ptr(s.SubmittedFlag),
		IsCorrect:         ptr(s.IsCorrect),
		CreatedAt:         ptr(s.CreatedAt.Format(time.RFC3339)),
		Username:          ptr(s.Username),
		TeamName:          ptr(s.TeamName),
		ChallengeTitle:    ptr(s.ChallengeTitle),
		ChallengeCategory: ptr(s.ChallengeCategory),
	}
	if s.TeamID != nil {
		res.TeamID = ptr(s.TeamID.String())
	}
	if s.IP != "" {
		res.IP = ptr(s.IP)
	}
	return res
}

func FromSubmissionList(items []*entity.SubmissionWithDetails, total int64, page, perPage int) openapi.ResponseSubmissionListResponse {
	resItems := make([]openapi.ResponseSubmissionResponse, len(items))
	for i, item := range items {
		resItems[i] = FromSubmission(item)
	}
	return openapi.ResponseSubmissionListResponse{
		Items:   &resItems,
		Total:   ptr(int(total)),
		Page:    ptr(page),
		PerPage: ptr(perPage),
	}
}

func FromSubmissionStats(stats *entity.SubmissionStats) openapi.ResponseSubmissionStatsResponse {
	return openapi.ResponseSubmissionStatsResponse{
		Total:     ptr(stats.Total),
		Correct:   ptr(stats.Correct),
		Incorrect: ptr(stats.Incorrect),
	}
}
