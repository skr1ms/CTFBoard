package entity

import (
	"github.com/google/uuid"
)

// FlagHashRegexSentinel is stored in Challenge.FlagHash for regex-mode
// challenges to distinguish them from hash-mode challenges without requiring a
// separate DB column.
const FlagHashRegexSentinel = "REGEX_CHALLENGE"

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
	IsHidden          bool      `json:"is_hidden"`
	IsRegex           bool      `json:"is_regex"`
	IsCaseInsensitive bool      `json:"is_case_insensitive"`
	FlagRegex         string    `json:"-"`
	FlagFormatRegex   *string   `json:"flag_format_regex,omitempty"`
}
