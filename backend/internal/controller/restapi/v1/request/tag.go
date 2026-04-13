package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type tagConstraints struct {
	Name  string `validate:"required"`
	Color string `validate:"omitempty,hex_color"`
}

func ValidateCreateTagRequest(req *openapi.CreateTagRequest, v validator.Validator) error {
	c := tagConstraints{Name: req.Name, Color: lo.FromPtrOr(req.Color, "")}

	return ValidateConstraints(v, &c)
}

func ValidateUpdateTagRequest(req *openapi.UpdateTagRequest, v validator.Validator) error {
	c := tagConstraints{Name: req.Name, Color: lo.FromPtrOr(req.Color, "")}

	return ValidateConstraints(v, &c)
}

func CreateTagRequestToParams(req *openapi.CreateTagRequest) (name, color string, err error) {
	return req.Name, lo.FromPtrOr(req.Color, ""), nil
}

func UpdateTagRequestToParams(req *openapi.UpdateTagRequest) (name, color string, err error) {
	return req.Name, lo.FromPtrOr(req.Color, ""), nil
}
