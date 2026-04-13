package domain

import (
	"time"

	"github.com/google/uuid"
)

type (
	// AuditAction is a string-backed enum for the type of administrative action recorded.
	AuditAction string
	// AuditEntityType is a string-backed enum identifying which domain entity was affected.
	AuditEntityType string
)

const (
	// AuditActionUpdate records a modification to an entity.
	AuditActionUpdate AuditAction = "update"
	// AuditActionDelete records the removal of an entity.
	AuditActionDelete AuditAction = "delete"
	// AuditActionImportErase records a bulk erase triggered during a backup import.
	AuditActionImportErase AuditAction = "import_erase"

	// AuditEntityChallenge identifies the challenge entity in audit records.
	AuditEntityChallenge AuditEntityType = "challenge"
	// AuditEntityCompetition identifies the competition entity in audit records.
	AuditEntityCompetition AuditEntityType = "competition"
	// AuditEntityAppSettings identifies the app-level settings entity in audit records.
	AuditEntityAppSettings AuditEntityType = "app_settings"
	// AuditEntityBackup identifies a backup operation in audit records.
	AuditEntityBackup AuditEntityType = "backup"
	// AuditEntityUser identifies a user entity in audit records.
	AuditEntityUser AuditEntityType = "user"
)

// AuditLog records an administrative action performed on a domain entity.
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
