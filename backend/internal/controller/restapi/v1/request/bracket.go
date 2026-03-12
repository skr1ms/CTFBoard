package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxBracketNameLength        = 200
	maxBracketDescriptionLength = 500
)

func CreateBracketRequestToParams(req *openapi.CreateBracketRequest) (name, description string, isDefault bool, err error) {
	if len(req.Name) > maxBracketNameLength {
		return "", "", false, helper.NewValidationErrorf("name too long")
	}
	desc := derefOr(req.Description, "")
	if len(desc) > maxBracketDescriptionLength {
		return "", "", false, helper.NewValidationErrorf("description too long")
	}
	return req.Name, desc, derefOr(req.IsDefault, false), nil
}

func UpdateBracketRequestToParams(req *openapi.UpdateBracketRequest) (name, description string, isDefault bool, err error) {
	if len(req.Name) > maxBracketNameLength {
		return "", "", false, helper.NewValidationErrorf("name too long")
	}
	desc := derefOr(req.Description, "")
	if len(desc) > maxBracketDescriptionLength {
		return "", "", false, helper.NewValidationErrorf("description too long")
	}
	return req.Name, desc, derefOr(req.IsDefault, false), nil
}

func SetTeamBracketRequestToParams(req *openapi.SetTeamBracketRequest) (*uuid.UUID, error) {
	if req.BracketID == nil {
		return nil, nil
	}
	id := *req.BracketID
	if id == uuid.Nil {
		return nil, helper.NewValidationErrorf("invalid bracket_id")
	}
	return &id, nil
}
