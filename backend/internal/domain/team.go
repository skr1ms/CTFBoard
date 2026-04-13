package domain

import (
	"time"

	"github.com/google/uuid"
)

// Team represents a group of participants. IsSolo is true for auto-created wrapper teams
// that represent individual players in team-based competition modes. IsBanned and IsHidden
// exclude the team from public views. DeletedAt implements soft deletion.
type Team struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	InviteToken          uuid.UUID  `json:"-"`
	InviteTokenExpiresAt *time.Time `json:"-"`
	CaptainID            uuid.UUID  `json:"captain_id"`
	BracketID            *uuid.UUID `json:"bracket_id,omitempty"`
	IsSolo               bool       `json:"is_solo"`
	IsAutoCreated        bool       `json:"is_auto_created"`
	IsBanned             bool       `json:"is_banned"`
	BannedAt             *time.Time `json:"banned_at,omitempty"`
	BannedReason         *string    `json:"banned_reason,omitempty"`
	IsHidden             bool       `json:"is_hidden"`
	AvatarURL            *string    `json:"avatar_url,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	DeletedAt            *time.Time `json:"-"`
}
