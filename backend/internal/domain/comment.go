package domain

import (
	"time"

	"github.com/google/uuid"
)

// Comment is a text message posted by a user on a challenge discussion thread.
type Comment struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
