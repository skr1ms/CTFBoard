package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type createFieldConstraints struct {
	Name       string   `validate:"required,max=100"`
	FieldType  string   `validate:"required,oneof=text number select boolean"`
	EntityType string   `validate:"required,oneof=user team"`
	Options    []string `validate:"max=100,dive,max=500"`
}

type updateFieldConstraints struct {
	Name      string   `validate:"required,max=100"`
	FieldType string   `validate:"required,oneof=text number select boolean"`
	Options   []string `validate:"max=100,dive,max=500"`
}

func ValidateCreateFieldRequest(req *openapi.CreateFieldRequest, v validator.Validator) error {
	c := createFieldConstraints{Name: req.Name, FieldType: string(req.FieldType), EntityType: string(req.EntityType), Options: lo.FromPtrOr(req.Options, nil)}

	return ValidateConstraints(v, &c)
}

func ValidateUpdateFieldRequest(req *openapi.UpdateFieldRequest, v validator.Validator) error {
	c := updateFieldConstraints{Name: req.Name, FieldType: string(req.FieldType), Options: lo.FromPtrOr(req.Options, nil)}

	return ValidateConstraints(v, &c)
}

func CreateFieldRequestToParams(req *openapi.CreateFieldRequest) (name string, fieldType domain.FieldType, entityType domain.EntityType, required bool, options []string, orderIndex int, err error) {
	return req.Name, domain.FieldType(req.FieldType), domain.EntityType(req.EntityType),
		lo.FromPtrOr(req.Required, false),
		lo.FromPtrOr(req.Options, nil),
		lo.FromPtrOr(req.OrderIndex, 0),
		nil
}

func UpdateFieldRequestToParams(req *openapi.UpdateFieldRequest) (name string, fieldType domain.FieldType, required bool, options []string, orderIndex int, err error) {
	return req.Name, domain.FieldType(req.FieldType),
		lo.FromPtrOr(req.Required, false),
		lo.FromPtrOr(req.Options, nil),
		lo.FromPtrOr(req.OrderIndex, 0),
		nil
}
