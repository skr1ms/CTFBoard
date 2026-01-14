package response

import "time"

type RegisterResponse struct {
	Id       string    `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	CreateAt time.Time `json:"created_at"`
}

type MeResponse struct {
	Id       string    `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	TeamId   *string   `json:"team_id"`
	CreateAt time.Time `json:"created_at"`
}

type UserProfileResponse struct {
	Id       string          `json:"id"`
	Username string          `json:"username"`
	TeamId   *string         `json:"team_id"`
	CreateAt time.Time       `json:"created_at"`
	Solves   []SolveResponse `json:"solves"`
}

type UserResponse struct {
	Id       string  `json:"id"`
	Username string  `json:"username"`
	TeamId   *string `json:"team_id"`
	Role     string  `json:"role"`
}

type SolveResponse struct {
	Id          string    `json:"id"`
	ChallengeId string    `json:"challenge_id"`
	SolvedAt    time.Time `json:"solved_at"`
}
