package domain

import (
	"time"

	"github.com/google/uuid"
)

// FlagHashRegexSentinel is stored in FlagHash when the challenge uses regex-based flag
// matching rather than a hashed comparison.
const FlagHashRegexSentinel = "REGEX_CHALLENGE"

const (
	// ChallengeStateVisible marks a challenge as publicly visible to participants.
	ChallengeStateVisible = "visible"
	// ChallengeStateHidden marks a challenge as invisible to participants.
	ChallengeStateHidden = "hidden"
	// ChallengeStateLocked marks a challenge as visible but not solvable until prerequisites are met.
	ChallengeStateLocked = "locked"
)

const (
	// ChallengeTypeStandard is a challenge with a fixed flag.
	ChallengeTypeStandard = "standard"
	// ChallengeTypeDynamic is a challenge with dynamic (per-team) scoring.
	ChallengeTypeDynamic = "dynamic"
)

// ChallengeStateOrDefault returns state if it is one of visible/hidden/locked, otherwise returns "hidden".
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

// Challenge is the main challenge entity holding scoring parameters, flag verification
// configuration, and display metadata.
type Challenge struct {
	ID                uuid.UUID     `json:"id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Category          string        `json:"category"`
	Points            int           `json:"points"`
	InitialValue      int           `json:"initial_value"`
	MinValue          int           `json:"min_value"`
	Decay             int           `json:"decay"`
	SolveCount        int           `json:"solve_count"`
	FlagHash          string        `json:"-"`
	ConnectionInfo    string        `json:"-"`
	MaxAttempts       int           `json:"-"`
	MaxAttemptsWindow time.Duration `json:"-"`
	Position          int           `json:"-"`
	State             string        `json:"-"`
	IsRegex           bool          `json:"is_regex"`
	IsCaseInsensitive bool          `json:"is_case_insensitive"`
	FlagRegex         *string       `json:"-"`
	FlagFormatRegex   *string       `json:"flag_format_regex,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}
