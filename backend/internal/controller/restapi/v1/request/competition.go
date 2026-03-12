package request

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// singletonCompetitionID is the fixed ID for the one competition row in the DB.
const singletonCompetitionID = 1

func ValidateCompetitionTimes(req *openapi.UpdateCompetitionRequest) error {
	startTime, endTime, freezeTime := req.StartTime, req.EndTime, req.FreezeTime
	if endTime != nil && startTime != nil && endTime.Before(*startTime) {
		return helper.NewValidationErrorf("end_time must be after start_time")
	}
	if freezeTime != nil && endTime != nil && !freezeTime.Before(*endTime) {
		return helper.NewValidationErrorf("freeze_time must be before end_time")
	}
	if freezeTime != nil && startTime != nil && freezeTime.Before(*startTime) {
		return helper.NewValidationErrorf("freeze_time must be after start_time")
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

func UpdateCompetitionRequestToEntity(req *openapi.UpdateCompetitionRequest) *entity.Competition {
	var mode entity.CompetitionMode
	if req.Mode != nil {
		mode = entity.CompetitionMode(*req.Mode)
	}
	comp := &entity.Competition{
		ID:                           singletonCompetitionID,
		Name:                         req.Name,
		StartTime:                    truncateTimePtr(req.StartTime),
		EndTime:                      truncateTimePtr(req.EndTime),
		FreezeTime:                   truncateTimePtr(req.FreezeTime),
		IsPaused:                     derefOr(req.IsPaused, false),
		IsPublic:                     derefOr(req.IsPublic, false),
		FlagRegex:                    req.FlagRegex,
		AllowTeamSwitch:              derefOr(req.AllowTeamSwitch, false),
		KeepScoreboardFrozenAfterEnd: derefOr(req.KeepScoreboardFrozenAfterEnd, false),
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
