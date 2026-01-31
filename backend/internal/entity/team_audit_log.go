package entity

import (
	"time"

	"github.com/google/uuid"
)

type TeamAuditAction string

const (
	TeamActionCreated         TeamAuditAction = "created"
	TeamActionJoined          TeamAuditAction = "joined"
	TeamActionLeft            TeamAuditAction = "left"
	TeamActionCaptainTransfer TeamAuditAction = "captain_transferred"
	TeamActionDeleted         TeamAuditAction = "deleted"
	TeamActionMemberKicked    TeamAuditAction = "member_kicked"
)

type TeamAuditLog struct {
	ID        uuid.UUID       `json:"id"`
	TeamID    uuid.UUID       `json:"team_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Action    TeamAuditAction `json:"action"`
	Details   map[string]any  `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
