package entity

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	InviteToken uuid.UUID `json:"invite_token"`
	CaptainId   uuid.UUID `json:"captain_id"`
	CreatedAt   time.Time `json:"created_at"`
}
