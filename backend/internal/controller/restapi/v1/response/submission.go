package response

import (
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromSubmission(s *domain.SubmissionWithDetails) openapi.SubmissionResponse {
	res := openapi.SubmissionResponse{
		ID:                httputil.Ptr(s.ID.String()),
		UserID:            httputil.Ptr(s.UserID.String()),
		ChallengeID:       httputil.Ptr(s.ChallengeID.String()),
		SubmittedFlag:     httputil.Ptr(s.SubmittedFlag),
		IsCorrect:         httputil.Ptr(s.IsCorrect),
		CreatedAt:         httputil.Ptr(s.CreatedAt.Format(time.RFC3339)),
		Username:          httputil.Ptr(s.Username),
		TeamName:          httputil.Ptr(s.TeamName),
		ChallengeTitle:    httputil.Ptr(s.ChallengeTitle),
		ChallengeCategory: httputil.Ptr(s.ChallengeCategory),
	}
	if s.TeamID != nil {
		res.TeamID = httputil.Ptr(s.TeamID.String())
	}
	if s.IP != "" {
		res.IP = httputil.Ptr(s.IP)
	}
	return res
}

func FromSubmissionPublic(s *domain.SubmissionWithDetails) openapi.SubmissionResponse {
	res := openapi.SubmissionResponse{
		ID:                httputil.Ptr(s.ID.String()),
		UserID:            httputil.Ptr(s.UserID.String()),
		ChallengeID:       httputil.Ptr(s.ChallengeID.String()),
		IsCorrect:         httputil.Ptr(s.IsCorrect),
		CreatedAt:         httputil.Ptr(s.CreatedAt.Format(time.RFC3339)),
		Username:          httputil.Ptr(s.Username),
		TeamName:          httputil.Ptr(s.TeamName),
		ChallengeTitle:    httputil.Ptr(s.ChallengeTitle),
		ChallengeCategory: httputil.Ptr(s.ChallengeCategory),
	}
	if s.TeamID != nil {
		res.TeamID = httputil.Ptr(s.TeamID.String())
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
		Total:     httputil.Ptr(stats.Total),
		Correct:   httputil.Ptr(stats.Correct),
		Incorrect: httputil.Ptr(stats.Incorrect),
	}
}
