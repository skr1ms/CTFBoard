package domain

import (
	"time"

	"github.com/google/uuid"
)

// Hint is a purchasable clue for a challenge; unlocking it deducts Cost points from the team.
type Hint struct {
	ID          uuid.UUID `json:"id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Cost        int       `json:"cost"`
	OrderIndex  int       `json:"order_index"`
}

// HintUnlock records that a team has paid for and unlocked a specific hint.
type HintUnlock struct {
	ID           uuid.UUID  `json:"id"`
	HintID       uuid.UUID  `json:"hint_id"`
	TeamID       uuid.UUID  `json:"team_id"`
	UnlockedAt   time.Time  `json:"unlocked_at"`
	BannedTeamID *uuid.UUID `json:"banned_team_id,omitempty"`
}

// HintUnlockWithDetails embeds HintUnlock data with resolved challenge and cost fields for responses.
type HintUnlockWithDetails struct {
	ID          uuid.UUID `json:"id"`
	HintID      uuid.UUID `json:"hint_id"`
	TeamID      uuid.UUID `json:"team_id"`
	UnlockedAt  time.Time `json:"unlocked_at"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	HintCost    int       `json:"hint_cost"`
}
