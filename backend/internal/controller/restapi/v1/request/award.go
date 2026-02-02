package request

import (
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func CreateAwardRequestToParams(req *openapi.RequestCreateAwardRequest) (teamID uuid.UUID, value int, description string, err error) {
	teamID, err = uuid.Parse(req.TeamID)
	return teamID, req.Value, req.Description, err
}
