package entity

import (
	"time"

	"github.com/google/uuid"
)

type (
	AuditAction     string
	AuditEntityType string
)

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
	AuditEntityAppSettings AuditEntityType = "app_settings"
)

type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	UserID     *uuid.UUID      `json:"user_id"`
	Action     AuditAction     `json:"action"`
	EntityType AuditEntityType `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	IP         string          `json:"ip"`
	Details    map[string]any  `json:"details,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
