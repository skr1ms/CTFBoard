package entity

import (
	"time"

	"github.com/google/uuid"
)

type FieldType string

const (
	FieldTypeText    FieldType = "text"
	FieldTypeNumber  FieldType = "number"
	FieldTypeSelect  FieldType = "select"
	FieldTypeBoolean FieldType = "boolean"
)

type EntityType string

const (
	EntityTypeUser EntityType = "user"
	EntityTypeTeam EntityType = "team"
)

type Field struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	FieldType  FieldType  `json:"field_type"`
	EntityType EntityType `json:"entity_type"`
	Required   bool       `json:"required"`
	Options    []string   `json:"options,omitempty"`
	OrderIndex int        `json:"order_index"`
	CreatedAt  time.Time  `json:"created_at"`
}

type FieldValue struct {
	ID        uuid.UUID `json:"id"`
	FieldID   uuid.UUID `json:"field_id"`
	EntityID  uuid.UUID `json:"entity_id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
