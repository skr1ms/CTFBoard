package domain

import (
	"time"

	"github.com/google/uuid"
)

// Topic is an organizer-facing taxonomy entry for grouping challenges beyond public tags.
type Topic struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
