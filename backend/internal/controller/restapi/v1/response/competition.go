package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromCompetition(c *domain.Competition) openapi.CompetitionResponse {
	return openapi.CompetitionResponse{
		ID:                           new(c.ID),
		Name:                         new(c.Name),
		StartTime:                    timePtr(c.StartTime),
		EndTime:                      timePtr(c.EndTime),
		FreezeTime:                   timePtr(c.FreezeTime),
		IsPaused:                     new(c.IsPaused),
		PausedAt:                     timePtr(c.PausedAt),
		IsPublic:                     new(c.IsPublic),
		KeepScoreboardFrozenAfterEnd: new(c.KeepScoreboardFrozenAfterEnd),
		Status:                       new(string(c.GetStatus())),
		Mode:                         new(string(c.Mode)),
		AllowTeamSwitch:              new(c.AllowTeamSwitch),
		MinTeamSize:                  new(c.MinTeamSize),
		MaxTeamSize:                  new(c.MaxTeamSize),
		FlagRegex:                    c.FlagRegex,
	}
}

func FromCompetitionStatus(c *domain.Competition) openapi.CompetitionStatusResponse {
	return openapi.CompetitionStatusResponse{
		Status:                       new(string(c.GetStatus())),
		Name:                         new(c.Name),
		StartTime:                    timePtr(c.StartTime),
		EndTime:                      timePtr(c.EndTime),
		FreezeTime:                   timePtr(c.FreezeTime),
		PausedAt:                     timePtr(c.PausedAt),
		KeepScoreboardFrozenAfterEnd: new(c.KeepScoreboardFrozenAfterEnd),
		SubmissionAllowed:            new(c.IsSubmissionAllowed()),
	}
}
