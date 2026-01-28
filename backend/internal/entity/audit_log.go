package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditAction string
type AuditEntityType string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionBan    AuditAction = "ban"
	AuditActionUnban  AuditAction = "unban"

	AuditEntityChallenge   AuditEntityType = "challenge"
	AuditEntityCompetition AuditEntityType = "competition"
	AuditEntityTeam        AuditEntityType = "team"
	AuditEntityUser        AuditEntityType = RoleUser
)

type AuditLog struct {
	Id         uuid.UUID       `json:"id"`
	UserId     *uuid.UUID      `json:"user_id"`
	Action     AuditAction     `json:"action"`
	EntityType AuditEntityType `json:"entity_type"`
	EntityId   string          `json:"entity_id"`
	IP         string          `json:"ip"`
	Details    map[string]any  `json:"details,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
