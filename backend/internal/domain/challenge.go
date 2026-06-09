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
	// ChallengeStateLocked marks a challenge as visible to participants but blocked from submissions.
	ChallengeStateLocked = "locked"
)

const (
	// SolutionStateHidden hides the official solution from participants.
	SolutionStateHidden = "hidden"
	// SolutionStateSolvedOnly shows the official solution after the viewer's team solves the challenge.
	SolutionStateSolvedOnly = "solved_only"
	// SolutionStateAfterEvent shows the official solution after the competition has ended.
	SolutionStateAfterEvent = "after_event"
	// SolutionStateAdminOnly keeps the official solution available only through admin flows.
	SolutionStateAdminOnly = "admin_only"
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

// SolutionStateOrDefault returns state if it is recognized, otherwise returns solved_only.
func SolutionStateOrDefault(state string) string {
	switch state {
	case SolutionStateHidden, SolutionStateSolvedOnly, SolutionStateAfterEvent, SolutionStateAdminOnly:
		return state
	default:
		return SolutionStateSolvedOnly
	}
}

// CanViewSolution evaluates the player/admin visibility contract for solution content
// and writeup files. The caller supplies the already-computed solve and competition state.
func CanViewSolution(state string, solved, eventEnded, writeupEnabled, isAdmin bool) bool {
	if isAdmin {
		return true
	}

	if !writeupEnabled {
		return false
	}

	switch SolutionStateOrDefault(state) {
	case SolutionStateSolvedOnly:
		return solved
	case SolutionStateAfterEvent:
		return eventEnded
	default:
		return false
	}
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
	State       string
	Files       []*File
}

// ChallengeSolutionEntry is a flattened solution row used in list responses.
type ChallengeSolutionEntry struct {
	ChallengeID       uuid.UUID
	ChallengeTitle    string
	ChallengeCategory string
	Content           string
	State             string
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
	Attribution       string        `json:"attribution"`
	ConnectionInfo    string        `json:"connection_info"`
	MaxAttempts       int           `json:"max_attempts"`
	MaxAttemptsWindow time.Duration `json:"max_attempts_window"`
	Position          int           `json:"position"`
	NextChallengeID   *uuid.UUID    `json:"next_id,omitempty"`
	State             string        `json:"state"`
	IsRegex           bool          `json:"is_regex"`
	IsCaseInsensitive bool          `json:"is_case_insensitive"`
	FlagRegex         *string       `json:"-"`
	FlagFormatRegex   *string       `json:"flag_format_regex,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}
