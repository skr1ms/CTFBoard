package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
)

type RegisterResponse struct {
	Id       string    `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	CreateAt time.Time `json:"created_at"`
}

func FromUserForRegister(u *entity.User) RegisterResponse {
	return RegisterResponse{
		Id:       u.Id.String(),
		Username: u.Username,
		Email:    u.Email,
		CreateAt: u.CreatedAt,
	}
}

type MeResponse struct {
	Id       string    `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	TeamId   *string   `json:"team_id"`
	CreateAt time.Time `json:"created_at"`
}

func FromUserForMe(u *entity.User) MeResponse {
	var teamIdStr *string
	if u.TeamId != nil {
		s := u.TeamId.String()
		teamIdStr = &s
	}
	return MeResponse{
		Id:       u.Id.String(),
		Username: u.Username,
		Email:    u.Email,
		TeamId:   teamIdStr,
		CreateAt: u.CreatedAt,
	}
}

type UserProfileResponse struct {
	Id       string          `json:"id"`
	Username string          `json:"username"`
	TeamId   *string         `json:"team_id"`
	CreateAt time.Time       `json:"created_at"`
	Solves   []SolveResponse `json:"solves"`
}

func FromUserProfile(up *usecase.UserProfile) UserProfileResponse {
	var teamIdStr *string
	if up.User.TeamId != nil {
		s := up.User.TeamId.String()
		teamIdStr = &s
	}

	solves := make([]SolveResponse, 0, len(up.Solves))
	for _, solve := range up.Solves {
		solves = append(solves, FromSolve(solve))
	}

	return UserProfileResponse{
		Id:       up.User.Id.String(),
		Username: up.User.Username,
		TeamId:   teamIdStr,
		CreateAt: up.User.CreatedAt,
		Solves:   solves,
	}
}

type UserResponse struct {
	Id       string  `json:"id"`
	Username string  `json:"username"`
	TeamId   *string `json:"team_id"`
	Role     string  `json:"role"`
}

func FromUser(u *entity.User) UserResponse {
	var teamIdStr *string
	if u.TeamId != nil {
		s := u.TeamId.String()
		teamIdStr = &s
	}
	return UserResponse{
		Id:       u.Id.String(),
		Username: u.Username,
		TeamId:   teamIdStr,
		Role:     u.Role,
	}
}

type SolveResponse struct {
	Id          string    `json:"id"`
	ChallengeId string    `json:"challenge_id"`
	SolvedAt    time.Time `json:"solved_at"`
}

func FromSolve(s *entity.Solve) SolveResponse {
	return SolveResponse{
		Id:          s.Id.String(),
		ChallengeId: s.ChallengeId.String(),
		SolvedAt:    s.SolvedAt,
	}
}
