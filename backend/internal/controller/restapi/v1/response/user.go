package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
)

func ptr[T any](v T) *T {
	return &v
}

func FromUserForRegister(u *entity.User) openapi.ResponseRegisterResponse {
	return openapi.ResponseRegisterResponse{
		ID:        ptr(u.ID.String()),
		Username:  ptr(u.Username),
		Email:     ptr(u.Email),
		CreatedAt: ptr(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserForMe(u *entity.User) openapi.ResponseMeResponse {
	var teamIDStr *string
	if u.TeamID != nil {
		s := u.TeamID.String()
		teamIDStr = &s
	}
	return openapi.ResponseMeResponse{
		ID:        ptr(u.ID.String()),
		Username:  ptr(u.Username),
		Email:     ptr(u.Email),
		TeamID:    teamIDStr,
		CreatedAt: ptr(u.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserProfile(up *user.UserProfile) openapi.ResponseUserProfileResponse {
	var teamIDStr *string
	if up.User.TeamID != nil {
		s := up.User.TeamID.String()
		teamIDStr = &s
	}

	solves := make([]openapi.ResponseSolveResponse, 0, len(up.Solves))
	for _, solve := range up.Solves {
		solves = append(solves, FromSolve(solve))
	}

	return openapi.ResponseUserProfileResponse{
		ID:        ptr(up.User.ID.String()),
		Username:  ptr(up.User.Username),
		TeamID:    teamIDStr,
		CreatedAt: ptr(up.User.CreatedAt.Format(time.RFC3339)),
		Solves:    &solves,
	}
}

func FromUser(u *entity.User) openapi.ResponseUserResponse {
	var teamIDStr *string
	if u.TeamID != nil {
		s := u.TeamID.String()
		teamIDStr = &s
	}
	return openapi.ResponseUserResponse{
		ID:       ptr(u.ID.String()),
		Username: ptr(u.Username),
		TeamID:   teamIDStr,
		Role:     ptr(u.Role), // Keeping it safe if type changes, but linter says redundant if Role is string.
	}
}

func FromSolve(s *entity.Solve) openapi.ResponseSolveResponse {
	return openapi.ResponseSolveResponse{
		ID:          ptr(s.ID.String()),
		ChallengeID: ptr(s.ChallengeID.String()),
		SolvedAt:    ptr(s.SolvedAt.Format(time.RFC3339)),
	}
}
