package domain

import (
	"time"

	"github.com/google/uuid"
)

// Bracket is a competitive division that teams can be assigned to for separate scoring tracks.
type Bracket struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
}
