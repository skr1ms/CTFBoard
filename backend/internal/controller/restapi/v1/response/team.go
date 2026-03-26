package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromTeam(t *domain.Team) openapi.TeamResponse {
	if t == nil {
		return openapi.TeamResponse{}
	}

	return openapi.TeamResponse{
		ID:          new(t.ID.String()),
		Name:        new(t.Name),
		InviteToken: new(t.InviteToken.String()),
		CaptainID:   new(t.CaptainID.String()),
		CreatedAt:   new(t.CreatedAt.Format(time.RFC3339)),
	}
}

func FromTeamWithoutToken(t *domain.Team) openapi.TeamResponse {
	if t == nil {
		return openapi.TeamResponse{}
	}

	return openapi.TeamResponse{
		ID:        new(t.ID.String()),
		Name:      new(t.Name),
		CaptainID: new(t.CaptainID.String()),
		CreatedAt: new(t.CreatedAt.Format(time.RFC3339)),
	}
}

func FromTeamWithMembers(t *domain.Team, members []*domain.User, minTeamSize int, meetsMinSize bool) openapi.TeamWithMembersResponse {
	if t == nil {
		return openapi.TeamWithMembersResponse{}
	}

	memberResponses := make([]openapi.UserResponse, 0, len(members))
	for _, member := range members {
		if member != nil {
			memberResponses = append(memberResponses, FromUser(member))
		}
	}

	res := openapi.TeamWithMembersResponse{
		ID:           new(t.ID.String()),
		Name:         new(t.Name),
		InviteToken:  new(t.InviteToken.String()),
		CaptainID:    new(t.CaptainID.String()),
		CreatedAt:    new(t.CreatedAt.Format(time.RFC3339)),
		Members:      &memberResponses,
		IsBanned:     new(t.IsBanned),
		MinTeamSize:  new(minTeamSize),
		MeetsMinSize: new(meetsMinSize),
	}

	if t.BannedAt != nil {
		res.BannedAt = new(t.BannedAt.Format(time.RFC3339))
	}

	if t.BannedReason != nil {
		res.BannedReason = t.BannedReason
	}

	return res
}

func FromTeamList(teams []*domain.Team, total int64, page, perPage int) openapi.TeamListResponse {
	data, meta := BuildListResponse(teams, FromTeamWithoutToken, total, page, perPage)

	return openapi.TeamListResponse{Data: &data, Meta: meta}
}

func FromAdminTeam(t *domain.Team, memberCount *int) openapi.AdminTeamResponse {
	res := openapi.AdminTeamResponse{
		ID:          new(t.ID.String()),
		Name:        new(t.Name),
		CaptainID:   new(t.CaptainID.String()),
		IsSolo:      new(t.IsSolo),
		IsBanned:    new(t.IsBanned),
		IsHidden:    new(t.IsHidden),
		MemberCount: memberCount,
		CreatedAt:   new(t.CreatedAt),
	}
	if t.BracketID != nil {
		res.BracketID = new(t.BracketID.String())
	}

	if t.BannedReason != nil {
		res.BannedReason = t.BannedReason
	}

	return res
}

func FromAdminTeamList(teams []*domain.Team, total int64, page, perPage int) openapi.AdminTeamListResponse {
	data, meta := BuildListResponse(teams, func(t *domain.Team) openapi.AdminTeamResponse { return FromAdminTeam(t, nil) }, total, page, perPage)

	return openapi.AdminTeamListResponse{Data: &data, Meta: meta}
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
		SolveCount:      new(a.SolveCount),
		Points:          new(a.Points),
		HintUnlockCount: new(a.HintUnlockCount),
		AwardsTotal:     new(a.AwardsTotal),
	}
}
