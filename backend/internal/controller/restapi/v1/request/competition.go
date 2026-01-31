package request

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

type UpdateCompetitionRequest struct {
	Name            string     `json:"name" Validate:"required,min=1,max=100"`
	StartTime       *time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	FreezeTime      *time.Time `json:"freeze_time"`
	IsPaused        bool       `json:"is_paused"`
	IsPublic        bool       `json:"is_public"`
	FlagRegex       *string    `json:"flag_regex"`
	AllowTeamSwitch bool       `json:"allow_team_switch"`
	Mode            string     `json:"mode"`
}

func (r *UpdateCompetitionRequest) ToCompetition(ID int) *entity.Competition {
	return &entity.Competition{
		ID:              ID,
		Name:            r.Name,
		StartTime:       r.StartTime,
		EndTime:         r.EndTime,
		FreezeTime:      r.FreezeTime,
		IsPaused:        r.IsPaused,
		IsPublic:        r.IsPublic,
		FlagRegex:       r.FlagRegex,
		AllowTeamSwitch: r.AllowTeamSwitch,
		Mode:            r.Mode,
	}
}
