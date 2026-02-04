package entity

import (
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	TeamID        *uuid.UUID `json:"team_id,omitempty"`
	ChallengeID   uuid.UUID  `json:"challenge_id"`
	SubmittedFlag string     `json:"submitted_flag"`
	IsCorrect     bool       `json:"is_correct"`
	IP            string     `json:"ip,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type SubmissionWithDetails struct {
	Submission
	Username          string `json:"username,omitempty"`
	TeamName          string `json:"team_name,omitempty"`
	ChallengeTitle    string `json:"challenge_title,omitempty"`
	ChallengeCategory string `json:"challenge_category,omitempty"`
}

type SubmissionStats struct {
	Total     int `json:"total"`
	Correct   int `json:"correct"`
	Incorrect int `json:"incorrect"`
}
