package request

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateBracketRequestToParams(req *openapi.CreateBracketRequest) (name, description string, isDefault bool, err error) {
	return req.Name, lo.FromPtrOr(req.Description, ""), lo.FromPtrOr(req.IsDefault, false), nil
}

func UpdateBracketRequestToParams(req *openapi.UpdateBracketRequest) (name, description string, isDefault bool, err error) {
	return req.Name, lo.FromPtrOr(req.Description, ""), lo.FromPtrOr(req.IsDefault, false), nil
}

func SetTeamBracketRequestToParams(req *openapi.SetTeamBracketRequest) (*uuid.UUID, error) {
	if req.BracketID == nil {
		return nil, nil
	}

	id := *req.BracketID
	if id == uuid.Nil {
		return nil, apperr.NewValidationErrorf("invalid bracket_id")
	}

	return &id, nil
}
