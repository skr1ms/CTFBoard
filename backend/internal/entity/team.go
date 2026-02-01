package entity

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	InviteToken   uuid.UUID  `json:"invite_token"`
	CaptainID     uuid.UUID  `json:"captain_id"`
	IsSolo        bool       `json:"is_solo"`
	IsAutoCreated bool       `json:"is_auto_created"`
	IsBanned      bool       `json:"is_banned"`
	BannedAt      *time.Time `json:"banned_at,omitempty"`
	BannedReason  *string    `json:"banned_reason,omitempty"`
	IsHidden      bool       `json:"is_hidden"`
	CreatedAt     time.Time  `json:"created_at"`
}
