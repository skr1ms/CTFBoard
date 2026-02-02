package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromTeam(t *entity.Team) openapi.ResponseTeamResponse {
	return openapi.ResponseTeamResponse{
		ID:          ptr(t.ID.String()),
		Name:        ptr(t.Name),
		InviteToken: ptr(t.InviteToken.String()),
		CaptainID:   ptr(t.CaptainID.String()),
		CreatedAt:   ptr(t.CreatedAt.Format(time.RFC3339)),
	}
}

func FromTeamWithoutToken(t *entity.Team) openapi.ResponseTeamResponse {
	return openapi.ResponseTeamResponse{
		ID:        ptr(t.ID.String()),
		Name:      ptr(t.Name),
		CaptainID: ptr(t.CaptainID.String()),
		CreatedAt: ptr(t.CreatedAt.Format(time.RFC3339)),
	}
}

func FromTeamWithMembers(t *entity.Team, members []*entity.User) openapi.ResponseTeamWithMembersResponse {
	memberResponses := make([]openapi.ResponseUserResponse, 0, len(members))
	for _, member := range members {
		memberResponses = append(memberResponses, FromUser(member))
	}

	resp := openapi.ResponseTeamWithMembersResponse{
		ID:          ptr(t.ID.String()),
		Name:        ptr(t.Name),
		InviteToken: ptr(t.InviteToken.String()),
		CaptainID:   ptr(t.CaptainID.String()),
		CreatedAt:   ptr(t.CreatedAt.Format(time.RFC3339)),
		Members:     &memberResponses,
		IsBanned:    ptr(t.IsBanned),
	}

	if t.BannedAt != nil {
		resp.BannedAt = ptr(t.BannedAt.Format(time.RFC3339))
	}
	if t.BannedReason != nil {
		resp.BannedReason = t.BannedReason
	}

	return resp
}
