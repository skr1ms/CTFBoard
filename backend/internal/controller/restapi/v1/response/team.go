package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromTeam(t *entity.Team) openapi.TeamResponse {
	return openapi.TeamResponse{
		ID:          ptr(t.ID.String()),
		Name:        ptr(t.Name),
		InviteToken: ptr(t.InviteToken.String()),
		CaptainID:   ptr(t.CaptainID.String()),
		CreatedAt:   ptr(t.CreatedAt.Format(time.RFC3339)),
	}
}

func FromTeamWithoutToken(t *entity.Team) openapi.TeamResponse {
	return openapi.TeamResponse{
		ID:        ptr(t.ID.String()),
		Name:      ptr(t.Name),
		CaptainID: ptr(t.CaptainID.String()),
		CreatedAt: ptr(t.CreatedAt.Format(time.RFC3339)),
	}
}

func FromTeamWithMembers(t *entity.Team, members []*entity.User, minTeamSize int, meetsMinSize bool) openapi.TeamWithMembersResponse {
	memberResponses := make([]openapi.UserResponse, len(members))
	for i, member := range members {
		memberResponses[i] = FromUser(member)
	}

	res := openapi.TeamWithMembersResponse{
		ID:           ptr(t.ID.String()),
		Name:         ptr(t.Name),
		InviteToken:  ptr(t.InviteToken.String()),
		CaptainID:    ptr(t.CaptainID.String()),
		CreatedAt:    ptr(t.CreatedAt.Format(time.RFC3339)),
		Members:      &memberResponses,
		IsBanned:     ptr(t.IsBanned),
		MinTeamSize:  ptr(minTeamSize),
		MeetsMinSize: ptr(meetsMinSize),
	}

	if t.BannedAt != nil {
		res.BannedAt = ptr(t.BannedAt.Format(time.RFC3339))
	}
	if t.BannedReason != nil {
		res.BannedReason = t.BannedReason
	}

	return res
}

func FromTeamList(teams []*entity.Team, total int64, page, perPage int) openapi.TeamListResponse {
	items := make([]openapi.TeamResponse, len(teams))
	for i, t := range teams {
		items[i] = FromTeamWithoutToken(t)
	}
	return openapi.TeamListResponse{
		Data: &items,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}

func FromAdminTeam(t *entity.Team, memberCount *int) openapi.AdminTeamResponse {
	res := openapi.AdminTeamResponse{
		ID:          ptr(t.ID.String()),
		Name:        ptr(t.Name),
		CaptainID:   ptr(t.CaptainID.String()),
		IsSolo:      ptr(t.IsSolo),
		IsBanned:    ptr(t.IsBanned),
		IsHidden:    ptr(t.IsHidden),
		MemberCount: memberCount,
		CreatedAt:   ptr(t.CreatedAt),
	}
	if t.BracketID != nil {
		res.BracketID = ptr(t.BracketID.String())
	}
	if t.BannedReason != nil {
		res.BannedReason = t.BannedReason
	}
	return res
}

func FromAdminTeamList(teams []*entity.Team, total int64, page, perPage int) openapi.AdminTeamListResponse {
	items := make([]openapi.AdminTeamResponse, len(teams))
	for i, t := range teams {
		items[i] = FromAdminTeam(t, nil)
	}
	return openapi.AdminTeamListResponse{
		Data: &items,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}

func FromTeamInvite(inviteToken string) openapi.TeamInviteResponse {
	return openapi.TeamInviteResponse{InviteToken: &inviteToken}
}

func FromConfirmationRequired(reason string, affected *openapi.AffectedData) openapi.ConfirmationRequired {
	return openapi.ConfirmationRequired{
		Reason:       reason,
		AffectedData: affected,
	}
}

func FromHiddenStatus(hidden bool) openapi.HiddenStatus {
	return openapi.HiddenStatus{Hidden: hidden}
}

func FromAffectedData(a *usecase.TeamCreateAffectedData) *openapi.AffectedData {
	if a == nil {
		return nil
	}
	return &openapi.AffectedData{
		SolveCount:      ptr(a.SolveCount),
		Points:          ptr(a.Points),
		HintUnlockCount: ptr(a.HintUnlockCount),
		AwardsTotal:     ptr(a.AwardsTotal),
	}
}
