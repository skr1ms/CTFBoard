package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromSubmission(s *entity.SubmissionWithDetails) openapi.SubmissionResponse {
	res := openapi.SubmissionResponse{
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

func FromSubmissionList(items []*entity.SubmissionWithDetails, total int64, page, perPage int) openapi.SubmissionListResponse {
	data := make([]openapi.SubmissionResponse, len(items))
	for i, item := range items {
		data[i] = FromSubmission(item)
	}
	return openapi.SubmissionListResponse{
		Data: &data,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}

func FromSubmissionStats(stats *entity.SubmissionStats) openapi.SubmissionStatsResponse {
	return openapi.SubmissionStatsResponse{
		Total:     ptr(stats.Total),
		Correct:   ptr(stats.Correct),
		Incorrect: ptr(stats.Incorrect),
	}
}
