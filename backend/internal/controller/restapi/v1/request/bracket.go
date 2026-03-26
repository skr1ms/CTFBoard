package request

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type createBracketConstraints struct {
	Name        string `validate:"required,max=200"`
	Description string `validate:"max=500"`
}

type updateBracketConstraints struct {
	Name        string `validate:"required,max=200"`
	Description string `validate:"max=500"`
}

func ValidateCreateBracketRequest(req *openapi.CreateBracketRequest, v validator.Validator) error {
	c := createBracketConstraints{Name: req.Name, Description: lo.FromPtrOr(req.Description, "")}

	return ValidateConstraints(v, &c)
}

func ValidateUpdateBracketRequest(req *openapi.UpdateBracketRequest, v validator.Validator) error {
	c := updateBracketConstraints{Name: req.Name, Description: lo.FromPtrOr(req.Description, "")}

	return ValidateConstraints(v, &c)
}

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
		return nil, httperr.NewValidationErrorf("invalid bracket_id")
	}

	return &id, nil
}
