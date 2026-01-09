package entity

import "time"

type User struct {
	Id           string    `json:"id"`
	TeamId       *string   `json:"team_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
