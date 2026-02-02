package request

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func UpdateCompetitionRequestToEntity(req *openapi.RequestUpdateCompetitionRequest, id int) *entity.Competition {
	var isPaused, isPublic, allowTeamSwitch bool
	if req.IsPaused != nil {
		isPaused = *req.IsPaused
	}
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	if req.AllowTeamSwitch != nil {
		allowTeamSwitch = *req.AllowTeamSwitch
	}
	var mode string
	if req.Mode != nil {
		mode = *req.Mode
	}
	return &entity.Competition{
		ID:              id,
		Name:            req.Name,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		FreezeTime:      req.FreezeTime,
		IsPaused:        isPaused,
		IsPublic:        isPublic,
		FlagRegex:       req.FlagRegex,
		AllowTeamSwitch: allowTeamSwitch,
		Mode:            mode,
	}
}
