package entity

import "time"

type Team struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	InviteToken string    `json:"invite_token"`
	CaptainId   string    `json:"captain_id"`
	CreatedAt   time.Time `json:"created_at"`
}
