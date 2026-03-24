package response

import (
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromCompetition(c *domain.Competition) openapi.CompetitionResponse {
	var startTime, endTime, freezeTime, pausedAt *string
	if c.StartTime != nil {
		startTime = httputil.Ptr(c.StartTime.Format(time.RFC3339))
	}
	if c.EndTime != nil {
		endTime = httputil.Ptr(c.EndTime.Format(time.RFC3339))
	}
	if c.FreezeTime != nil {
		freezeTime = httputil.Ptr(c.FreezeTime.Format(time.RFC3339))
	}
	if c.PausedAt != nil {
		pausedAt = httputil.Ptr(c.PausedAt.Format(time.RFC3339))
	}
	return openapi.CompetitionResponse{
		ID:                           httputil.Ptr(c.ID),
		Name:                         httputil.Ptr(c.Name),
		StartTime:                    startTime,
		EndTime:                      endTime,
		FreezeTime:                   freezeTime,
		IsPaused:                     httputil.Ptr(c.IsPaused),
		PausedAt:                     pausedAt,
		IsPublic:                     httputil.Ptr(c.IsPublic),
		KeepScoreboardFrozenAfterEnd: httputil.Ptr(c.KeepScoreboardFrozenAfterEnd),
		Status:                       httputil.Ptr(string(c.GetStatus())),
		Mode:                         httputil.Ptr(string(c.Mode)),
	}
}

func FromCompetitionStatus(c *domain.Competition) openapi.CompetitionStatusResponse {
	var startTime, endTime, freezeTime, pausedAt *string
	if c.StartTime != nil {
		startTime = httputil.Ptr(c.StartTime.Format(time.RFC3339))
	}
	if c.EndTime != nil {
		endTime = httputil.Ptr(c.EndTime.Format(time.RFC3339))
	}
	if c.FreezeTime != nil {
		freezeTime = httputil.Ptr(c.FreezeTime.Format(time.RFC3339))
	}
	if c.PausedAt != nil {
		pausedAt = httputil.Ptr(c.PausedAt.Format(time.RFC3339))
	}
	return openapi.CompetitionStatusResponse{
		Status:                       httputil.Ptr(string(c.GetStatus())),
		Name:                         httputil.Ptr(c.Name),
		StartTime:                    startTime,
		EndTime:                      endTime,
		FreezeTime:                   freezeTime,
		PausedAt:                     pausedAt,
		KeepScoreboardFrozenAfterEnd: httputil.Ptr(c.KeepScoreboardFrozenAfterEnd),
		SubmissionAllowed:            httputil.Ptr(c.IsSubmissionAllowed()),
	}
}
