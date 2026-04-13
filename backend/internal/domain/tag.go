package domain

import "github.com/google/uuid"

// Tag is a colored label that can be attached to challenges for categorization and filtering.
type Tag struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color string    `json:"color"`
}
