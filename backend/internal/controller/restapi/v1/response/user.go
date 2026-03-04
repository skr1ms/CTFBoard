package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/google/uuid"
)

func FromUserForRegister(u *entity.User) openapi.RegisterResponse {
	return openapi.RegisterResponse{
		ID:        ptr(u.ID.String()),
		Username:  ptr(u.Username),
		Email:     ptr(u.Email),
		CreatedAt: ptr(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserForMe(u *entity.User) openapi.MeResponse {
	var teamIDStr *string
	if u.TeamID != nil {
		teamIDStr = ptr(u.TeamID.String())
	}
	return openapi.MeResponse{
		ID:        ptr(u.ID.String()),
		Username:  ptr(u.Username),
		Email:     ptr(u.Email),
		Role:      ptr(u.Role),
		TeamID:    teamIDStr,
		CreatedAt: ptr(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserProfile(up *usecase.UserProfile) openapi.UserProfileResponse {
	var teamIDStr *string
	if up.User.TeamID != nil {
		teamIDStr = ptr(up.User.TeamID.String())
	}

	solves := make([]openapi.SolveResponse, len(up.Solves))
	for i, solve := range up.Solves {
		solves[i] = FromSolve(solve)
	}

	return openapi.UserProfileResponse{
		ID:        ptr(up.User.ID.String()),
		Username:  ptr(up.User.Username),
		TeamID:    teamIDStr,
		CreatedAt: ptr(up.User.CreatedAt.Format(time.RFC3339)),
		Solves:    &solves,
	}
}

func FromUser(u *entity.User) openapi.UserResponse {
	var teamIDStr *string
	if u.TeamID != nil {
		teamIDStr = ptr(u.TeamID.String())
	}
	return openapi.UserResponse{
		ID:       ptr(u.ID.String()),
		Username: ptr(u.Username),
		TeamID:   teamIDStr,
		Role:     ptr(u.Role),
	}
}

func FromSolve(s *entity.Solve) openapi.SolveResponse {
	return openapi.SolveResponse{
		ID:          ptr(s.ID.String()),
		ChallengeID: ptr(s.ChallengeID.String()),
		SolvedAt:    ptr(s.SolvedAt),
	}
}

func FromTokenPair(p *jwt.TokenPair) openapi.TokenPair {
	return openapi.TokenPair{
		AccessToken:      ptr(p.AccessToken),
		AccessExpiresAt:  ptr(int(p.AccessExpiresAt)),
		RefreshToken:     ptr(p.RefreshToken),
		RefreshExpiresAt: ptr(int(p.RefreshExpiresAt)),
	}
}

func FromUserList(users []*entity.User, total int64, page, perPage int) openapi.UserListResponse {
	items := make([]openapi.UserResponse, len(users))
	for i, u := range users {
		items[i] = FromUser(u)
	}
	return openapi.UserListResponse{
		Data: &items,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}

func FromAdminUser(u *entity.User) openapi.AdminUserResponse {
	var teamIDStr *string
	if u.TeamID != nil {
		teamIDStr = ptr(u.TeamID.String())
	}
	return openapi.AdminUserResponse{
		ID:           ptr(u.ID.String()),
		Username:     ptr(u.Username),
		Email:        ptr(u.Email),
		Role:         ptr(u.Role),
		TeamID:       teamIDStr,
		IsVerified:   ptr(u.IsVerified),
		CreatedAt:    ptr(u.CreatedAt),
		IsBanned:     ptr(u.IsBanned),
		BannedAt:     u.BannedAt,
		BannedReason: u.BannedReason,
	}
}

func FromAdminUserList(users []*entity.User, total int64, page, perPage int) openapi.AdminUserListResponse {
	items := make([]openapi.AdminUserResponse, len(users))
	for i, u := range users {
		items[i] = FromAdminUser(u)
	}
	return openapi.AdminUserListResponse{
		Data: &items,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}

func FromAdminUserSlice(users []*entity.User) []openapi.AdminUserResponse {
	items := make([]openapi.AdminUserResponse, len(users))
	for i, u := range users {
		items[i] = FromAdminUser(u)
	}
	return items
}

func FromSolveWithDetailsList(solves []*entity.SolveWithDetails) []openapi.SolveWithDetailsResponse {
	res := make([]openapi.SolveWithDetailsResponse, len(solves))
	for i, s := range solves {
		res[i] = openapi.SolveWithDetailsResponse{
			ID:                ptr(s.ID.String()),
			UserID:            ptr(s.UserID.String()),
			ChallengeID:       ptr(s.ChallengeID.String()),
			SolvedAt:          ptr(s.SolvedAt),
			Username:          ptr(s.Username),
			TeamName:          ptr(s.TeamName),
			ChallengeTitle:    ptr(s.ChallengeTitle),
			ChallengeCategory: ptr(s.ChallengeCategory),
			ChallengePoints:   ptr(s.ChallengePoints),
		}
		if s.TeamID != uuid.Nil {
			res[i].TeamID = ptr(s.TeamID.String())
		}
	}
	return res
}

func FromFailList(fails []*entity.SubmissionWithDetails, total int64, page, perPage int) openapi.FailListResponse {
	data := make([]openapi.SubmissionResponse, len(fails))
	for i, f := range fails {
		data[i] = FromSubmission(f)
	}
	return openapi.FailListResponse{
		Data: &data,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}
