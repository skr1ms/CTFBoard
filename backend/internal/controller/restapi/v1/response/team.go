package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

type TeamResponse struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	InviteToken string    `json:"invite_token,omitempty"`
	CaptainId   string    `json:"captain_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func FromTeam(t *entity.Team) TeamResponse {
	return TeamResponse{
		Id:          t.Id.String(),
		Name:        t.Name,
		InviteToken: t.InviteToken.String(),
		CaptainId:   t.CaptainId.String(),
		CreatedAt:   t.CreatedAt,
	}
}

func FromTeamWithoutToken(t *entity.Team) TeamResponse {
	return TeamResponse{
		Id:        t.Id.String(),
		Name:      t.Name,
		CaptainId: t.CaptainId.String(),
		CreatedAt: t.CreatedAt,
	}
}

type TeamWithMembersResponse struct {
	Id          string         `json:"id"`
	Name        string         `json:"name"`
	InviteToken string         `json:"invite_token"`
	CaptainId   string         `json:"captain_id"`
	CreatedAt   time.Time      `json:"created_at"`
	Members     []UserResponse `json:"members"`
}

func FromTeamWithMembers(t *entity.Team, members []*entity.User) TeamWithMembersResponse {
	memberResponses := make([]UserResponse, 0, len(members))
	for _, member := range members {
		memberResponses = append(memberResponses, FromUser(member))
	}

	return TeamWithMembersResponse{
		Id:          t.Id.String(),
		Name:        t.Name,
		InviteToken: t.InviteToken.String(),
		CaptainId:   t.CaptainId.String(),
		CreatedAt:   t.CreatedAt,
		Members:     memberResponses,
	}
}
