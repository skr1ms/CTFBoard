package domain

import (
	"time"

	"github.com/google/uuid"
)

// TeamAuditAction is a string-backed enum for team lifecycle events recorded in the audit log.
type TeamAuditAction string

const (
	// TeamActionCreated records that the team was created.
	TeamActionCreated TeamAuditAction = "created"
	// TeamActionJoined records that a user joined the team.
	TeamActionJoined TeamAuditAction = "joined"
	// TeamActionLeft records that a user voluntarily left the team.
	TeamActionLeft TeamAuditAction = "left"
	// TeamActionCaptainTransfer records a captain role transfer to another member.
	TeamActionCaptainTransfer TeamAuditAction = "captain_transferred"
	// TeamActionDeleted records that the team was deleted.
	TeamActionDeleted TeamAuditAction = "deleted"
	// TeamActionMemberKicked records that a member was removed by the captain.
	TeamActionMemberKicked TeamAuditAction = "member_kicked"
	// TeamActionMemberBanned records that a member was banned from the team.
	TeamActionMemberBanned TeamAuditAction = "member_banned"
	// TeamActionBanned records that the entire team was banned by an admin.
	TeamActionBanned TeamAuditAction = "banned"
	// TeamActionUnbanned records that the team's ban was lifted by an admin.
	TeamActionUnbanned TeamAuditAction = "unbanned"

	TeamAuditDetailReason = "reason"
)

// TeamAuditLog records a team lifecycle event. The Data field carries an action-specific
// JSON payload used for audit display and undo operations.
type TeamAuditLog struct {
	ID        uuid.UUID       `json:"id"`
	TeamID    uuid.UUID       `json:"team_id"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	Action    TeamAuditAction `json:"action"`
	Details   map[string]any  `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
