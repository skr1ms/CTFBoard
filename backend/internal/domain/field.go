package domain

import (
	"time"

	"github.com/google/uuid"
)

// FieldType is a string-backed enum for the value type of a custom registration field
// Valid values: text, number, select, boolean, json.
type FieldType string

const (
	// FieldTypeText is a free-form text input field.
	FieldTypeText FieldType = "text"
	// FieldTypeNumber is a numeric input field.
	FieldTypeNumber FieldType = "number"
	// FieldTypeSelect is a single-choice field with predefined options.
	FieldTypeSelect FieldType = "select"
	// FieldTypeBoolean is a true/false toggle field.
	FieldTypeBoolean FieldType = "boolean"
	// FieldTypeJSON is an arbitrary JSON value field.
	FieldTypeJSON FieldType = "json"
)

// IsValid returns true if ft is one of the recognized field types.
func (ft FieldType) IsValid() bool {
	switch ft {
	case FieldTypeText, FieldTypeNumber, FieldTypeSelect, FieldTypeBoolean, FieldTypeJSON:
		return true
	}

	return false
}

// EntityType is a string-backed enum identifying which entity type a custom field is attached to
// Valid values: user, team.
type EntityType string

const (
	// EntityTypeUser associates the field with user profiles.
	EntityTypeUser EntityType = "user"
	// EntityTypeTeam associates the field with team profiles.
	EntityTypeTeam EntityType = "team"
)

// IsValid returns true if et is one of the recognized entity types (user or team).
func (et EntityType) IsValid() bool {
	switch et {
	case EntityTypeUser, EntityTypeTeam:
		return true
	}

	return false
}

// Field defines a custom registration field for users or teams.
type Field struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	FieldType   FieldType  `json:"field_type"`
	EntityType  EntityType `json:"entity_type"`
	Required    bool       `json:"required"`
	Public      bool       `json:"public"`
	Editable    bool       `json:"editable"`
	Options     []string   `json:"options,omitempty"`
	OrderIndex  int        `json:"order_index"`
	CreatedAt   time.Time  `json:"created_at"`
}

// FieldValue stores the value a specific entity (user or team) has submitted for a custom field.
type FieldValue struct {
	ID        uuid.UUID `json:"id"`
	FieldID   uuid.UUID `json:"field_id"`
	EntityID  uuid.UUID `json:"entity_id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
