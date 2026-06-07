package domain

import "time"

// Competition is the main competition entity, representing a single CTF event
// with its schedule, participation rules, and scoring configuration.
type Competition struct {
	ID                           int             `json:"id"`
	Name                         string          `json:"name"`
	StartTime                    *time.Time      `json:"start_time"`
	EndTime                      *time.Time      `json:"end_time"`
	FreezeTime                   *time.Time      `json:"freeze_time"`
	IsPaused                     bool            `json:"is_paused"`
	PausedAt                     *time.Time      `json:"paused_at,omitempty"`
	IsPublic                     bool            `json:"is_public"`
	FlagRegex                    *string         `json:"flag_regex,omitempty"`
	Mode                         CompetitionMode `json:"mode"`
	AllowTeamSwitch              bool            `json:"allow_team_switch"`
	MinTeamSize                  int             `json:"min_team_size"`
	MaxTeamSize                  int             `json:"max_team_size"`
	KeepScoreboardFrozenAfterEnd bool            `json:"keep_scoreboard_frozen_after_end"`
	CreatedAt                    time.Time       `json:"created_at"`
	UpdatedAt                    time.Time       `json:"updated_at"`
}

// CompetitionStatus is a string-backed enum representing the lifecycle state of a competition
// Valid values: not_started, active, paused, frozen, ended.
type CompetitionStatus string

const (
	// CompetitionStatusNotStarted indicates the competition has not yet begun (start time is in the future or unset).
	CompetitionStatusNotStarted CompetitionStatus = "not_started"
	// CompetitionStatusActive indicates the competition is currently running and accepting submissions.
	CompetitionStatusActive CompetitionStatus = "active"
	// CompetitionStatusPaused indicates the competition has been manually paused by an admin.
	CompetitionStatusPaused CompetitionStatus = "paused"
	// CompetitionStatusFrozen indicates the scoreboard is frozen but submissions are still accepted.
	CompetitionStatusFrozen CompetitionStatus = "frozen"
	// CompetitionStatusEnded indicates the competition has passed its end time.
	CompetitionStatusEnded CompetitionStatus = "ended"
)

// CompetitionMode is a string-backed enum controlling who may participate.
// Valid values: solo_only, teams_only.
type CompetitionMode string

const (
	// ModeSoloOnly restricts participation to individual players only.
	ModeSoloOnly CompetitionMode = "solo_only"
	// ModeTeamsOnly restricts participation to teams only.
	ModeTeamsOnly CompetitionMode = "teams_only"
	// DefaultCompetitionMode is the conservative default for serious CTF events.
	DefaultCompetitionMode CompetitionMode = ModeTeamsOnly
)

// IsValid returns true if m is one of the recognized competition modes.
func (m CompetitionMode) IsValid() bool {
	switch m {
	case ModeSoloOnly, ModeTeamsOnly:
		return true
	}

	return false
}

// AllowsSolo returns true if solo players may participate under this mode.
func (m CompetitionMode) AllowsSolo() bool {
	return m == ModeSoloOnly
}

// AllowsTeams returns true if teams may participate under this mode.
func (m CompetitionMode) AllowsTeams() bool {
	return m == ModeTeamsOnly
}

// getStatusAt evaluates the competition state machine at an arbitrary point in
// time. Precedence order: not yet started -> paused -> ended -> frozen -> active.
// It is the shared implementation for GetStatus and GetStatusAt.
func (c *Competition) getStatusAt(now time.Time) CompetitionStatus {
	if c.StartTime == nil || now.Before(*c.StartTime) {
		return CompetitionStatusNotStarted
	}

	if c.IsPaused {
		return CompetitionStatusPaused
	}

	if c.EndTime != nil && now.After(*c.EndTime) {
		return CompetitionStatusEnded
	}

	if c.FreezeTime != nil && now.After(*c.FreezeTime) {
		return CompetitionStatusFrozen
	}

	return CompetitionStatusActive
}

// isFreezeActiveAt returns true when the scoreboard freeze is in effect at
// the given time. The freeze is considered inactive when: the competition
// has not started, the freeze time has not been reached, the competition is
// paused, or the competition has ended and KeepScoreboardFrozenAfterEnd is
// false.
func (c *Competition) isFreezeActiveAt(now time.Time) bool {
	if c.StartTime == nil || now.Before(*c.StartTime) {
		return false
	}

	if c.FreezeTime == nil || !now.After(*c.FreezeTime) {
		return false
	}

	if !c.IsPaused && c.EndTime != nil && now.After(*c.EndTime) && !c.KeepScoreboardFrozenAfterEnd {
		return false
	}

	return true
}

// GetStatus returns the current competition status based on the current time.
func (c *Competition) GetStatus() CompetitionStatus {
	return c.getStatusAt(time.Now())
}

// GetStatusAt determines the competition status at the given point in time,
// evaluating start time, pause state, end time, and freeze time in order.
func (c *Competition) GetStatusAt(now time.Time) CompetitionStatus {
	return c.getStatusAt(now)
}

// IsFreezeActive returns true if the scoreboard freeze is currently in effect.
func (c *Competition) IsFreezeActive() bool {
	return c.isFreezeActiveAt(time.Now())
}

// IsSubmissionAllowed returns true when the competition is in Active or Frozen state.
func (c *Competition) IsSubmissionAllowed() bool {
	now := time.Now()
	status := c.getStatusAt(now)

	return status == CompetitionStatusActive || status == CompetitionStatusFrozen
}

// IsSubmissionAllowedAt returns true when the competition is in Active or Frozen state at the given time.
func (c *Competition) IsSubmissionAllowedAt(now time.Time) bool {
	status := c.getStatusAt(now)

	return status == CompetitionStatusActive || status == CompetitionStatusFrozen
}

// IsEffectivelyEnded returns true if the competition has ended, or if it is paused past its end time.
func (c *Competition) IsEffectivelyEnded(now time.Time) bool {
	if c.GetStatusAt(now) == CompetitionStatusEnded {
		return true
	}

	return c.IsPaused && c.EndTime != nil && now.After(*c.EndTime)
}
