package entity

import (
	"time"

	"github.com/google/uuid"
)

type Award struct {
	ID          uuid.UUID  `json:"id"`
	TeamID      uuid.UUID  `json:"team_id"`
	Value       int        `json:"value"`
	Description string     `json:"description"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
