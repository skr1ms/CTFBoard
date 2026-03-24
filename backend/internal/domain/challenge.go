package domain

import (
	"time"

	"github.com/google/uuid"
)

const FlagHashRegexSentinel = "REGEX_CHALLENGE"

const (
	ChallengeStateVisible = "visible"
	ChallengeStateHidden  = "hidden"
	ChallengeStateLocked  = "locked"
)

func ChallengeStateOrDefault(state string) string {
	if state == ChallengeStateHidden || state == ChallengeStateLocked || state == ChallengeStateVisible {
		return state
	}
	return ChallengeStateHidden
}

// ChallengeFlags holds the flag-verification data for a challenge.
type ChallengeFlags struct {
	FlagHash          string
	IsRegex           bool
	IsCaseInsensitive bool
	FlagRegex         *string
	FlagFormatRegex   *string
}

// ChallengeRequirement represents a prerequisite challenge that must be solved
// before unlocking the parent challenge.
type ChallengeRequirement struct {
	ChallengeID    uuid.UUID
	ChallengeTitle string
	Category       *string
}

// ChallengeSolution holds the solution content and associated files for a challenge.
type ChallengeSolution struct {
	ChallengeID uuid.UUID
	Content     string
	Files       []*File
}

// ChallengeSolutionEntry is a flattened solution row used in list responses.
type ChallengeSolutionEntry struct {
	ChallengeID       uuid.UUID
	ChallengeTitle    string
	ChallengeCategory string
	Content           string
	Files             []*File
}

// ChallengeWithSolved bundles a challenge with a per-viewer solve flag.
type ChallengeWithSolved struct {
	Challenge *Challenge
	Solved    bool
}

type Challenge struct {
	ID                uuid.UUID `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	Points            int       `json:"points"`
	InitialValue      int       `json:"initial_value"`
	MinValue          int       `json:"min_value"`
	Decay             int       `json:"decay"`
	SolveCount        int       `json:"solve_count"`
	FlagHash          string    `json:"-"`
	ConnectionInfo    string    `json:"-"`
	MaxAttempts       int       `json:"-"`
	Position          int       `json:"-"`
	State             string    `json:"-"`
	IsRegex           bool      `json:"is_regex"`
	IsCaseInsensitive bool      `json:"is_case_insensitive"`
	FlagRegex         *string   `json:"-"`
	FlagFormatRegex   *string   `json:"flag_format_regex,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
