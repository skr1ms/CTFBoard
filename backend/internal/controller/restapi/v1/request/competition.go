package request

import (
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
	if freezeTime != nil && endTime != nil && freezeTime.After(*endTime) {
		return helper.NewValidationErrorf("freeze_time must be before end_time")
	}
	if freezeTime != nil && startTime != nil && freezeTime.Before(*startTime) {
		return helper.NewValidationErrorf("freeze_time must be after start_time")
	}
	return nil
}

func UpdateCompetitionRequestToEntity(req *openapi.UpdateCompetitionRequest) *entity.Competition {
	var mode entity.CompetitionMode
	if req.Mode != nil {
		mode = entity.CompetitionMode(*req.Mode)
	}
	return &entity.Competition{
		ID:              singletonCompetitionID,
		Name:            req.Name,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		FreezeTime:      req.FreezeTime,
		IsPaused:        derefOr(req.IsPaused, false),
		IsPublic:        derefOr(req.IsPublic, false),
		FlagRegex:       req.FlagRegex,
		AllowTeamSwitch: derefOr(req.AllowTeamSwitch, false),
		Mode:            mode,
	}
}
