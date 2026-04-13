package request

import (
	"time"

	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// singletonCompetitionID is the fixed ID for the one competition row in the DB.
const singletonCompetitionID = 1

func ValidateCompetitionTimes(req *openapi.UpdateCompetitionRequest) error {
	startTime, endTime, freezeTime := req.StartTime, req.EndTime, req.FreezeTime
	if endTime != nil && startTime != nil && endTime.Before(*startTime) {
		return apperr.NewValidationErrorf("end_time must be after start_time")
	}

	if freezeTime != nil && endTime != nil && !freezeTime.Before(*endTime) {
		return apperr.NewValidationErrorf("freeze_time must be before end_time")
	}

	if freezeTime != nil && startTime != nil && freezeTime.Before(*startTime) {
		return apperr.NewValidationErrorf("freeze_time must be after start_time")
	}

	return nil
}

func truncateTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	tr := t.Truncate(time.Second)

	return &tr
}

func UpdateCompetitionOptionalsFromRequest(req *openapi.UpdateCompetitionRequest) *usecase.CompetitionUpdateOptionals {
	return &usecase.CompetitionUpdateOptionals{
		IsPaused:                     req.IsPaused,
		IsPublic:                     req.IsPublic,
		AllowTeamSwitch:              req.AllowTeamSwitch,
		MinTeamSize:                  req.MinTeamSize,
		MaxTeamSize:                  req.MaxTeamSize,
		ClearFreezeTime:              req.ClearFreezeTime,
		ClearEndTime:                 req.ClearEndTime,
		KeepScoreboardFrozenAfterEnd: req.KeepScoreboardFrozenAfterEnd,
	}
}

func UpdateCompetitionRequestToEntity(req *openapi.UpdateCompetitionRequest) *domain.Competition {
	var mode domain.CompetitionMode

	if req.Mode != nil {
		mode = domain.CompetitionMode(*req.Mode)
	}

	comp := &domain.Competition{
		ID:                           singletonCompetitionID,
		Name:                         req.Name,
		StartTime:                    truncateTimePtr(req.StartTime),
		EndTime:                      truncateTimePtr(req.EndTime),
		FreezeTime:                   truncateTimePtr(req.FreezeTime),
		IsPaused:                     lo.FromPtrOr(req.IsPaused, false),
		IsPublic:                     lo.FromPtrOr(req.IsPublic, false),
		FlagRegex:                    req.FlagRegex,
		AllowTeamSwitch:              lo.FromPtrOr(req.AllowTeamSwitch, false),
		KeepScoreboardFrozenAfterEnd: lo.FromPtrOr(req.KeepScoreboardFrozenAfterEnd, false),
		Mode:                         mode,
	}
	if req.MinTeamSize != nil {
		comp.MinTeamSize = *req.MinTeamSize
	}

	if req.MaxTeamSize != nil {
		comp.MaxTeamSize = *req.MaxTeamSize
	}

	return comp
}
