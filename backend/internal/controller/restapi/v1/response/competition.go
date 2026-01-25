package response

import "github.com/skr1ms/CTFBoard/internal/entity"

type CompetitionResponse struct {
	Id         int     `json:"id"`
	Name       string  `json:"name"`
	StartTime  *string `json:"start_time"`
	EndTime    *string `json:"end_time"`
	FreezeTime *string `json:"freeze_time"`
	IsPaused   bool    `json:"is_paused"`
	IsPublic   bool    `json:"is_public"`
	Status     string  `json:"status"`
}

func FromCompetition(c *entity.Competition) CompetitionResponse {
	var startTime, endTime, freezeTime *string
	if c.StartTime != nil {
		s := c.StartTime.Format("2006-01-02T15:04:05Z07:00")
		startTime = &s
	}
	if c.EndTime != nil {
		s := c.EndTime.Format("2006-01-02T15:04:05Z07:00")
		endTime = &s
	}
	if c.FreezeTime != nil {
		s := c.FreezeTime.Format("2006-01-02T15:04:05Z07:00")
		freezeTime = &s
	}

	return CompetitionResponse{
		Id:         c.Id,
		Name:       c.Name,
		StartTime:  startTime,
		EndTime:    endTime,
		FreezeTime: freezeTime,
		IsPaused:   c.IsPaused,
		IsPublic:   c.IsPublic,
		Status:     string(c.GetStatus()),
	}
}

type CompetitionStatusResponse struct {
	Status            string  `json:"status"`
	Name              string  `json:"name"`
	StartTime         *string `json:"start_time"`
	EndTime           *string `json:"end_time"`
	SubmissionAllowed bool    `json:"submission_allowed"`
}

func FromCompetitionStatus(c *entity.Competition) CompetitionStatusResponse {
	var startTime, endTime *string
	if c.StartTime != nil {
		s := c.StartTime.Format("2006-01-02T15:04:05Z07:00")
		startTime = &s
	}
	if c.EndTime != nil {
		s := c.EndTime.Format("2006-01-02T15:04:05Z07:00")
		endTime = &s
	}

	return CompetitionStatusResponse{
		Status:            string(c.GetStatus()),
		Name:              c.Name,
		StartTime:         startTime,
		EndTime:           endTime,
		SubmissionAllowed: c.IsSubmissionAllowed(),
	}
}
