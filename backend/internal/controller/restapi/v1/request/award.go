package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateAwardRequestToParams(req *openapi.CreateAwardRequest) (teamID uuid.UUID, value int, description string, err error) {
	teamID, parseErr := uuid.Parse(req.TeamID)
	if parseErr != nil {
		return uuid.Nil, 0, "", apperr.NewValidationErrorf("invalid team_id")
	}

	return teamID, req.Value, req.Description, nil
}
