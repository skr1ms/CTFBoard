package entity

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	InviteToken   uuid.UUID `json:"invite_token"`
	CaptainID     uuid.UUID `json:"captain_id"`
	IsSolo        bool      `json:"is_solo"`
	IsAutoCreated bool      `json:"is_auto_created"`
	CreatedAt     time.Time `json:"created_at"`
}
