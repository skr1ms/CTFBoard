package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
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
		ID:        new(u.ID.String()),
		Username:  new(u.Username),
		Email:     new(u.Email),
		CreatedAt: new(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserForMe(u *domain.User) openapi.MeResponse {
	if u == nil {
		return openapi.MeResponse{}
	}

	var teamIDStr *string

	if u.TeamID != nil {
		teamIDStr = new(u.TeamID.String())
	}

	return openapi.MeResponse{
		ID:        new(u.ID.String()),
		Username:  new(u.Username),
		Email:     new(u.Email),
		Role:      new(string(u.Role)),
		TeamID:    teamIDStr,
		CreatedAt: new(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserProfile(up *usecase.UserProfile) openapi.UserProfileResponse {
	if up == nil || up.User == nil {
		return openapi.UserProfileResponse{}
	}

	var teamIDStr *string

	if up.User.TeamID != nil {
		teamIDStr = new(up.User.TeamID.String())
	}

	solves := make([]openapi.SolveResponse, len(up.Solves))
	for i, solve := range up.Solves {
		solves[i] = FromSolve(solve)
	}

	return openapi.UserProfileResponse{
		ID:        new(up.User.ID.String()),
		Username:  new(up.User.Username),
		TeamID:    teamIDStr,
		CreatedAt: new(up.User.CreatedAt.Format(time.RFC3339)),
		Solves:    &solves,
	}
}

func FromUser(u *domain.User) openapi.UserResponse {
	if u == nil {
		return openapi.UserResponse{}
	}

	var teamIDStr *string

	if u.TeamID != nil {
		teamIDStr = new(u.TeamID.String())
	}

	return openapi.UserResponse{
		ID:       new(u.ID.String()),
		Username: new(u.Username),
		TeamID:   teamIDStr,
		Role:     new(string(u.Role)),
	}
}

func FromSolve(s *domain.Solve) openapi.SolveResponse {
	return openapi.SolveResponse{
		ID:          new(s.ID.String()),
		ChallengeID: new(s.ChallengeID.String()),
		SolvedAt:    new(s.SolvedAt),
	}
}

func FromTokenPair(p *jwtkit.TokenPair) openapi.TokenPair {
	return openapi.TokenPair{
		AccessToken:      new(p.AccessToken),
		AccessExpiresAt:  new(int(p.AccessExpiresAt)),
		RefreshToken:     new(p.RefreshToken),
		RefreshExpiresAt: new(int(p.RefreshExpiresAt)),
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
		teamIDStr = new(u.TeamID.String())
	}

	return openapi.AdminUserResponse{
		ID:           new(u.ID.String()),
		Username:     new(u.Username),
		Email:        new(u.Email),
		Role:         new(string(u.Role)),
		TeamID:       teamIDStr,
		IsVerified:   new(u.IsVerified),
		CreatedAt:    new(u.CreatedAt),
		IsBanned:     new(u.IsBanned),
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
			ID:                new(s.ID.String()),
			UserID:            new(s.UserID.String()),
			ChallengeID:       new(s.ChallengeID.String()),
			SolvedAt:          new(s.SolvedAt),
			Username:          new(s.Username),
			TeamName:          new(s.TeamName),
			ChallengeTitle:    new(s.ChallengeTitle),
			ChallengeCategory: new(s.ChallengeCategory),
			ChallengePoints:   new(s.ChallengePoints),
		}
		if s.TeamID != uuid.Nil {
			res[i].TeamID = new(s.TeamID.String())
		}
	}

	return res
}

func FromFailListPublic(fails []*domain.SubmissionWithDetails, total int64, page, perPage int) openapi.FailListResponse {
	data, meta := BuildListResponse(fails, FromSubmissionPublic, total, page, perPage)

	return openapi.FailListResponse{Data: &data, Meta: meta}
}
