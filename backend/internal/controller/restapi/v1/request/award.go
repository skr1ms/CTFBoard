package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateAwardRequestToParams(req *openapi.CreateAwardRequest) (teamID uuid.UUID, value int, description string, err error) {
	teamID, parseErr := uuid.Parse(req.TeamID)
	if parseErr != nil {
		return uuid.Nil, 0, "", helper.NewValidationErrorf("invalid team_id")
	}
	if req.Value < 0 {
		return uuid.Nil, 0, "", helper.NewValidationErrorf("value must be >= 0")
	}
	return teamID, req.Value, req.Description, nil
}
