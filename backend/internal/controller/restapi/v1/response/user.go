package response

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

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
		CreatedAt: timePtr(&u.CreatedAt),
	}
}

func FromUserMe(me *usecase.UserMe) openapi.MeResponse {
	if me == nil {
		return openapi.MeResponse{}
	}

	return FromUserForMe(me.User, me.CustomFields)
}

func FromUserForMe(u *domain.User, customFields ...usecase.CustomFieldValues) openapi.MeResponse {
	if u == nil {
		return openapi.MeResponse{}
	}

	var teamIDStr *string

	if u.TeamID != nil {
		teamIDStr = new(u.TeamID.String())
	}

	hasPassword := u.PasswordHash != "" && u.PasswordHash != domain.OAuthOnlyPasswordSentinel

	resp := openapi.MeResponse{
		ID:          new(u.ID.String()),
		Username:    new(u.Username),
		Email:       new(u.Email),
		Role:        new(string(u.Role)),
		TeamID:      teamIDStr,
		CreatedAt:   timePtr(&u.CreatedAt),
		AvatarURL:   u.AvatarURL,
		HasPassword: &hasPassword,
	}

	if u.IsBanned {
		resp.BanStatus = &openapi.BanStatus{
			IsBanned: &u.IsBanned,
			Reason:   u.BannedReason,
			BannedAt: timePtr(u.BannedAt),
		}
	}

	if len(customFields) > 0 && len(customFields[0]) > 0 {
		resp.CustomFields = &customFields[0]
	}

	return resp
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

	resp := openapi.UserProfileResponse{
		ID:        new(up.User.ID.String()),
		Username:  new(up.User.Username),
		TeamID:    teamIDStr,
		CreatedAt: timePtr(&up.User.CreatedAt),
		Solves:    &solves,
		AvatarURL: up.User.AvatarURL,
	}

	if len(up.CustomFields) > 0 {
		resp.CustomFields = &up.CustomFields
	}

	return resp
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
		ID:        new(u.ID.String()),
		Username:  new(u.Username),
		TeamID:    teamIDStr,
		Role:      new(string(u.Role)),
		AvatarURL: u.AvatarURL,
	}
}

func FromSetupComplete(accessToken string, u *domain.User) openapi.SetupCompleteResponse {
	return openapi.SetupCompleteResponse{
		Token: accessToken,
		User:  FromUser(u),
	}
}

func FromSetupStatus(complete bool) openapi.SetupStatusResponse {
	return openapi.SetupStatusResponse{Complete: complete}
}

func FromSolve(s *domain.Solve) openapi.SolveResponse {
	return openapi.SolveResponse{
		ID:          new(s.ID.String()),
		ChallengeID: new(s.ChallengeID.String()),
		SolvedAt:    new(s.SolvedAt),
	}
}

func FromTokenPair(p *usecase.TokenPair) openapi.TokenPair {
	if p == nil {
		return openapi.TokenPair{}
	}

	return FromTokenValues(p.AccessToken, p.AccessExpiresAt)
}

func FromTokenValues(accessToken string, accessExpiresAt int64) openapi.TokenPair {
	return openapi.TokenPair{
		AccessToken:     new(accessToken),
		AccessExpiresAt: new(int(accessExpiresAt)),
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
		AvatarURL:    u.AvatarURL,
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

func FromFailListSelf(fails []*domain.SubmissionWithDetails, total int64, page, perPage int) openapi.FailListResponse {
	data, meta := BuildListResponse(fails, FromSubmissionSelf, total, page, perPage)

	return openapi.FailListResponse{Data: &data, Meta: meta}
}
