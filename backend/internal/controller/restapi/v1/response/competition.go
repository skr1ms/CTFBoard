package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromCompetition(c *entity.Competition) openapi.CompetitionResponse {
	var startTime, endTime, freezeTime *string
	if c.StartTime != nil {
		startTime = ptr(c.StartTime.Format(time.RFC3339))
	}
	if c.EndTime != nil {
		endTime = ptr(c.EndTime.Format(time.RFC3339))
	}
	if c.FreezeTime != nil {
		freezeTime = ptr(c.FreezeTime.Format(time.RFC3339))
	}
	return openapi.CompetitionResponse{
		ID:         ptr(c.ID),
		Name:       ptr(c.Name),
		StartTime:  startTime,
		EndTime:    endTime,
		FreezeTime: freezeTime,
		IsPaused:   ptr(c.IsPaused),
		IsPublic:   ptr(c.IsPublic),
		Status:     ptr(string(c.GetStatus())),
		Mode:       ptr(string(c.Mode)),
	}
}

func FromCompetitionStatus(c *entity.Competition) openapi.CompetitionStatusResponse {
	var startTime, endTime *string
	if c.StartTime != nil {
		startTime = ptr(c.StartTime.Format(time.RFC3339))
	}
	if c.EndTime != nil {
		endTime = ptr(c.EndTime.Format(time.RFC3339))
	}
	return openapi.CompetitionStatusResponse{
		Status:            ptr(string(c.GetStatus())),
		Name:              ptr(c.Name),
		StartTime:         startTime,
		EndTime:           endTime,
		SubmissionAllowed: ptr(c.IsSubmissionAllowed()),
	}
}
