package domain

import (
	"time"

	"github.com/google/uuid"
)

// Award is a bonus point grant given to a team by an admin. BannedTeamID is set when
// the award is preserved for scoring history after the team is soft-banned.
type Award struct {
	ID           uuid.UUID  `json:"id"`
	TeamID       uuid.UUID  `json:"team_id"`
	Value        int        `json:"value"`
	Description  string     `json:"description"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	BannedTeamID *uuid.UUID `json:"banned_team_id,omitempty"`
}
