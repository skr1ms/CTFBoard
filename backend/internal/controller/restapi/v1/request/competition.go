package request

import "time"

type UpdateCompetitionRequest struct {
	Name       string     `json:"name" validate:"required,min=1,max=100"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	FreezeTime *time.Time `json:"freeze_time"`
	IsPaused   bool       `json:"is_paused"`
	IsPublic   bool       `json:"is_public"`
}
