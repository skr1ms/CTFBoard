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
	Id        uuid.UUID              `json:"id"`
	TeamId    uuid.UUID              `json:"team_id"`
	UserId    uuid.UUID              `json:"user_id"`
	Action    TeamAuditAction        `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}
