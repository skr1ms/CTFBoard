package response

import (
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
		SubmissionType:    new(s.Type),
		CreatedAt:         timePtr(&s.CreatedAt),
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

// FromSubmissionPublic maps a submission to the public API response format.
// SubmittedFlag and IP are intentionally omitted to prevent flag leakage and
// to protect submitter privacy when the endpoint is accessible to regular users.
func FromSubmissionPublic(s *domain.SubmissionWithDetails) openapi.SubmissionResponse {
	res := openapi.SubmissionResponse{
		ID:                new(s.ID.String()),
		UserID:            new(s.UserID.String()),
		ChallengeID:       new(s.ChallengeID.String()),
		IsCorrect:         new(s.IsCorrect),
		SubmissionType:    new(s.Type),
		CreatedAt:         timePtr(&s.CreatedAt),
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

// FromSubmissionSelf maps a submission for the requester's own data
// (/users/me/fails, /teams/me/fails). Includes SubmittedFlag (no flag-leak
// risk on own data) but omits IP (PII-adjacent; not useful to the user).
func FromSubmissionSelf(s *domain.SubmissionWithDetails) openapi.SubmissionResponse {
	res := FromSubmissionPublic(s)
	res.SubmittedFlag = new(s.SubmittedFlag)

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
