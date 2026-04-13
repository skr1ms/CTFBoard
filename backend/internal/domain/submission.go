package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	// SubmissionTypeCorrect indicates the submitted flag matched the challenge answer.
	SubmissionTypeCorrect = "correct"
	// SubmissionTypeIncorrect indicates the submitted flag did not match.
	SubmissionTypeIncorrect = "incorrect"
	// SubmissionTypeRatelimited indicates the submission was rejected due to rate limiting.
	SubmissionTypeRatelimited = "ratelimited"
	// SubmissionTypeDiscard indicates the submission was discarded, e.g. after the competition ended.
	SubmissionTypeDiscard = "discard"
)

// SubmissionTypeFromCorrect returns "correct" if isCorrect is true, "incorrect" otherwise.
func SubmissionTypeFromCorrect(isCorrect bool) string {
	if isCorrect {
		return SubmissionTypeCorrect
	}

	return SubmissionTypeIncorrect
}

// Submission records a single flag submission attempt by a user.
type Submission struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	TeamID        *uuid.UUID `json:"team_id,omitempty"`
	ChallengeID   uuid.UUID  `json:"challenge_id"`
	SubmittedFlag string     `json:"submitted_flag"`
	IsCorrect     bool       `json:"is_correct"`
	Type          string     `json:"type"`
	IP            string     `json:"ip,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	BannedUserID  *uuid.UUID `json:"banned_user_id,omitempty"`
	BannedTeamID  *uuid.UUID `json:"banned_team_id,omitempty"`
}

// SubmissionWithDetails embeds Submission and adds resolved foreign-key fields
// for display in list and detail responses.
type SubmissionWithDetails struct {
	Submission

	Username          string `json:"username,omitempty"`
	TeamName          string `json:"team_name,omitempty"`
	ChallengeTitle    string `json:"challenge_title,omitempty"`
	ChallengeCategory string `json:"challenge_category,omitempty"`
}

// SubmissionStats holds aggregate submission counts for a challenge or competition scope.
type SubmissionStats struct {
	Total     int `json:"total"`
	Correct   int `json:"correct"`
	Incorrect int `json:"incorrect"`
}
