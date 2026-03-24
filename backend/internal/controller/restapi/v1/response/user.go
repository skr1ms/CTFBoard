package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromUserForRegister(u *domain.User) openapi.RegisterResponse {
	if u == nil {
		return openapi.RegisterResponse{}
	}
	return openapi.RegisterResponse{
		ID:        httputil.Ptr(u.ID.String()),
		Username:  httputil.Ptr(u.Username),
		Email:     httputil.Ptr(u.Email),
		CreatedAt: httputil.Ptr(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserForMe(u *domain.User) openapi.MeResponse {
	if u == nil {
		return openapi.MeResponse{}
	}
	var teamIDStr *string
	if u.TeamID != nil {
		teamIDStr = httputil.Ptr(u.TeamID.String())
	}
	return openapi.MeResponse{
		ID:        httputil.Ptr(u.ID.String()),
		Username:  httputil.Ptr(u.Username),
		Email:     httputil.Ptr(u.Email),
		Role:      httputil.Ptr(string(u.Role)),
		TeamID:    teamIDStr,
		CreatedAt: httputil.Ptr(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserProfile(up *usecase.UserProfile) openapi.UserProfileResponse {
	if up == nil || up.User == nil {
		return openapi.UserProfileResponse{}
	}
	var teamIDStr *string
	if up.User.TeamID != nil {
		teamIDStr = httputil.Ptr(up.User.TeamID.String())
	}

	solves := make([]openapi.SolveResponse, len(up.Solves))
	for i, solve := range up.Solves {
		solves[i] = FromSolve(solve)
	}

	return openapi.UserProfileResponse{
		ID:        httputil.Ptr(up.User.ID.String()),
		Username:  httputil.Ptr(up.User.Username),
		TeamID:    teamIDStr,
		CreatedAt: httputil.Ptr(up.User.CreatedAt.Format(time.RFC3339)),
		Solves:    &solves,
	}
}

func FromUser(u *domain.User) openapi.UserResponse {
	if u == nil {
		return openapi.UserResponse{}
	}
	var teamIDStr *string
	if u.TeamID != nil {
		teamIDStr = httputil.Ptr(u.TeamID.String())
	}
	return openapi.UserResponse{
		ID:       httputil.Ptr(u.ID.String()),
		Username: httputil.Ptr(u.Username),
		TeamID:   teamIDStr,
		Role:     httputil.Ptr(string(u.Role)),
	}
}

func FromSolve(s *domain.Solve) openapi.SolveResponse {
	return openapi.SolveResponse{
		ID:          httputil.Ptr(s.ID.String()),
		ChallengeID: httputil.Ptr(s.ChallengeID.String()),
		SolvedAt:    httputil.Ptr(s.SolvedAt),
	}
}

func FromTokenPair(p *jwtkit.TokenPair) openapi.TokenPair {
	return openapi.TokenPair{
		AccessToken:      httputil.Ptr(p.AccessToken),
		AccessExpiresAt:  httputil.Ptr(int(p.AccessExpiresAt)),
		RefreshToken:     httputil.Ptr(p.RefreshToken),
		RefreshExpiresAt: httputil.Ptr(int(p.RefreshExpiresAt)),
	}
}

func FromUserList(users []*domain.User, total int64, page, perPage int) openapi.UserListResponse {
	data, meta := BuildListResponse(users, FromUser, total, page, perPage)
	return openapi.UserListResponse{Data: &data, Meta: meta}
}

func FromAdminUser(u *domain.User) openapi.AdminUserResponse {
	if u == nil {
		return openapi.AdminUserResponse{}
	}
	var teamIDStr *string
	if u.TeamID != nil {
		teamIDStr = httputil.Ptr(u.TeamID.String())
	}
	return openapi.AdminUserResponse{
		ID:           httputil.Ptr(u.ID.String()),
		Username:     httputil.Ptr(u.Username),
		Email:        httputil.Ptr(u.Email),
		Role:         httputil.Ptr(string(u.Role)),
		TeamID:       teamIDStr,
		IsVerified:   httputil.Ptr(u.IsVerified),
		CreatedAt:    httputil.Ptr(u.CreatedAt),
		IsBanned:     httputil.Ptr(u.IsBanned),
		BannedAt:     u.BannedAt,
		BannedReason: u.BannedReason,
	}
}

func FromAdminUserList(users []*domain.User, total int64, page, perPage int) openapi.AdminUserListResponse {
	data, meta := BuildListResponse(users, FromAdminUser, total, page, perPage)
	return openapi.AdminUserListResponse{Data: &data, Meta: meta}
}

func FromAdminUserSlice(users []*domain.User) []openapi.AdminUserResponse {
	return lo.Map(users, func(user *domain.User, _ int) openapi.AdminUserResponse { return FromAdminUser(user) })
}

func FromSolveWithDetailsList(solves []*domain.SolveWithDetails) []openapi.SolveWithDetailsResponse {
	res := make([]openapi.SolveWithDetailsResponse, len(solves))
	for i, s := range solves {
		res[i] = openapi.SolveWithDetailsResponse{
			ID:                httputil.Ptr(s.ID.String()),
			UserID:            httputil.Ptr(s.UserID.String()),
			ChallengeID:       httputil.Ptr(s.ChallengeID.String()),
			SolvedAt:          httputil.Ptr(s.SolvedAt),
			Username:          httputil.Ptr(s.Username),
			TeamName:          httputil.Ptr(s.TeamName),
			ChallengeTitle:    httputil.Ptr(s.ChallengeTitle),
			ChallengeCategory: httputil.Ptr(s.ChallengeCategory),
			ChallengePoints:   httputil.Ptr(s.ChallengePoints),
		}
		if s.TeamID != uuid.Nil {
			res[i].TeamID = httputil.Ptr(s.TeamID.String())
		}
	}
	return res
}

func FromFailListPublic(fails []*domain.SubmissionWithDetails, total int64, page, perPage int) openapi.FailListResponse {
	data, meta := BuildListResponse(fails, FromSubmissionPublic, total, page, perPage)
	return openapi.FailListResponse{Data: &data, Meta: meta}
}
