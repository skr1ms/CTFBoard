package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromCompetition(c *domain.Competition) openapi.CompetitionResponse {
	var startTime, endTime, freezeTime, pausedAt *string

	if c.StartTime != nil {
		startTime = new(c.StartTime.Format(time.RFC3339))
	}

	if c.EndTime != nil {
		endTime = new(c.EndTime.Format(time.RFC3339))
	}

	if c.FreezeTime != nil {
		freezeTime = new(c.FreezeTime.Format(time.RFC3339))
	}

	if c.PausedAt != nil {
		pausedAt = new(c.PausedAt.Format(time.RFC3339))
	}

	return openapi.CompetitionResponse{
		ID:                           new(c.ID),
		Name:                         new(c.Name),
		StartTime:                    startTime,
		EndTime:                      endTime,
		FreezeTime:                   freezeTime,
		IsPaused:                     new(c.IsPaused),
		PausedAt:                     pausedAt,
		IsPublic:                     new(c.IsPublic),
		KeepScoreboardFrozenAfterEnd: new(c.KeepScoreboardFrozenAfterEnd),
		Status:                       new(string(c.GetStatus())),
		Mode:                         new(string(c.Mode)),
	}
}

func FromCompetitionStatus(c *domain.Competition) openapi.CompetitionStatusResponse {
	var startTime, endTime, freezeTime, pausedAt *string

	if c.StartTime != nil {
		startTime = new(c.StartTime.Format(time.RFC3339))
	}

	if c.EndTime != nil {
		endTime = new(c.EndTime.Format(time.RFC3339))
	}

	if c.FreezeTime != nil {
		freezeTime = new(c.FreezeTime.Format(time.RFC3339))
	}

	if c.PausedAt != nil {
		pausedAt = new(c.PausedAt.Format(time.RFC3339))
	}

	return openapi.CompetitionStatusResponse{
		Status:                       new(string(c.GetStatus())),
		Name:                         new(c.Name),
		StartTime:                    startTime,
		EndTime:                      endTime,
		FreezeTime:                   freezeTime,
		PausedAt:                     pausedAt,
		KeepScoreboardFrozenAfterEnd: new(c.KeepScoreboardFrozenAfterEnd),
		SubmissionAllowed:            new(c.IsSubmissionAllowed()),
	}
}
