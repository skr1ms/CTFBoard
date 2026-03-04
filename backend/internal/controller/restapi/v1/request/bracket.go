package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
)

func CreateBracketRequestToParams(req *openapi.CreateBracketRequest) (name, description string, isDefault bool) {
	return req.Name, derefOr(req.Description, ""), derefOr(req.IsDefault, false)
}

func UpdateBracketRequestToParams(req *openapi.UpdateBracketRequest) (name, description string, isDefault bool) {
	return req.Name, derefOr(req.Description, ""), derefOr(req.IsDefault, false)
}

func SetTeamBracketRequestToParams(req *openapi.SetTeamBracketRequest) *uuid.UUID {
	return req.BracketID
}
