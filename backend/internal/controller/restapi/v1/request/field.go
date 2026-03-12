package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxFieldNameLength   = 100
	maxFieldOptionLength = 500
	maxFieldOptionsCount = 100
)

func CreateFieldRequestToParams(req *openapi.CreateFieldRequest) (name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int, err error) {
	fieldType = entity.FieldType(req.FieldType)
	if !fieldType.IsValid() {
		return "", "", "", false, nil, 0, helper.NewValidationErrorf("invalid field_type")
	}
	entityType = entity.EntityType(req.EntityType)
	if !entityType.IsValid() {
		return "", "", "", false, nil, 0, helper.NewValidationErrorf("invalid entity_type")
	}
	if len(req.Name) > maxFieldNameLength {
		return "", "", "", false, nil, 0, helper.NewValidationErrorf("name too long")
	}
	opts := derefOr(req.Options, nil)
	if len(opts) > maxFieldOptionsCount {
		return "", "", "", false, nil, 0, helper.NewValidationErrorf("options count too large")
	}
	for _, o := range opts {
		if len(o) > maxFieldOptionLength {
			return "", "", "", false, nil, 0, helper.NewValidationErrorf("option value too long")
		}
	}
	return req.Name, fieldType, entityType,
		derefOr(req.Required, false),
		opts,
		derefOr(req.OrderIndex, 0),
		nil
}

func UpdateFieldRequestToParams(req *openapi.UpdateFieldRequest) (name string, fieldType entity.FieldType, required bool, options []string, orderIndex int, err error) {
	fieldType = entity.FieldType(req.FieldType)
	if !fieldType.IsValid() {
		return "", "", false, nil, 0, helper.NewValidationErrorf("invalid field_type")
	}
	if len(req.Name) > maxFieldNameLength {
		return "", "", false, nil, 0, helper.NewValidationErrorf("name too long")
	}
	opts := derefOr(req.Options, nil)
	if len(opts) > maxFieldOptionsCount {
		return "", "", false, nil, 0, helper.NewValidationErrorf("options count too large")
	}
	for _, o := range opts {
		if len(o) > maxFieldOptionLength {
			return "", "", false, nil, 0, helper.NewValidationErrorf("option value too long")
		}
	}
	return req.Name, fieldType,
		derefOr(req.Required, false),
		opts,
		derefOr(req.OrderIndex, 0),
		nil
}
