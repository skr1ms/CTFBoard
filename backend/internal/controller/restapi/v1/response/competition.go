package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromCompetition(c *entity.Competition) openapi.ResponseCompetitionResponse {
	var startTime, endTime, freezeTime *string
	if c.StartTime != nil {
		s := c.StartTime.Format(time.RFC3339)
		startTime = &s
	}
	if c.EndTime != nil {
		s := c.EndTime.Format(time.RFC3339)
		endTime = &s
	}
	if c.FreezeTime != nil {
		s := c.FreezeTime.Format(time.RFC3339)
		freezeTime = &s
	}
	status := string(c.GetStatus())
	id := c.ID
	name := c.Name
	mode := c.Mode
	return openapi.ResponseCompetitionResponse{
		ID:         &id,
		Name:       &name,
		StartTime:  startTime,
		EndTime:    endTime,
		FreezeTime: freezeTime,
		IsPaused:   &c.IsPaused,
		IsPublic:   &c.IsPublic,
		Status:     &status,
		Mode:       &mode,
	}
}

func FromCompetitionStatus(c *entity.Competition) openapi.ResponseCompetitionStatusResponse {
	var startTime, endTime *string
	if c.StartTime != nil {
		s := c.StartTime.Format(time.RFC3339)
		startTime = &s
	}
	if c.EndTime != nil {
		s := c.EndTime.Format(time.RFC3339)
		endTime = &s
	}
	status := string(c.GetStatus())
	name := c.Name
	submissionAllowed := c.IsSubmissionAllowed()
	return openapi.ResponseCompetitionStatusResponse{
		Status:            &status,
		Name:              &name,
		StartTime:         startTime,
		EndTime:           endTime,
		SubmissionAllowed: &submissionAllowed,
	}
}
