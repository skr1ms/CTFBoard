package entity

import "time"

type Competition struct {
	Id         int        `json:"id"`
	Name       string     `json:"name"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	FreezeTime *time.Time `json:"freeze_time"`
	IsPaused   bool       `json:"is_paused"`
	IsPublic   bool       `json:"is_public"`
	FlagRegex  *string    `json:"flag_regex,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CompetitionStatus string

const (
	CompetitionStatusNotStarted CompetitionStatus = "not_started"
	CompetitionStatusActive     CompetitionStatus = "active"
	CompetitionStatusPaused     CompetitionStatus = "paused"
	CompetitionStatusFrozen     CompetitionStatus = "frozen"
	CompetitionStatusEnded      CompetitionStatus = "ended"
)

func (c *Competition) GetStatus() CompetitionStatus {
	now := time.Now()

	if c.StartTime == nil || now.Before(*c.StartTime) {
		return CompetitionStatusNotStarted
	}

	if c.EndTime != nil && now.After(*c.EndTime) {
		return CompetitionStatusEnded
	}

	if c.IsPaused {
		return CompetitionStatusPaused
	}

	if c.FreezeTime != nil && now.After(*c.FreezeTime) {
		return CompetitionStatusFrozen
	}

	return CompetitionStatusActive
}

func (c *Competition) IsSubmissionAllowed() bool {
	status := c.GetStatus()
	return status == CompetitionStatusActive || status == CompetitionStatusFrozen
}
