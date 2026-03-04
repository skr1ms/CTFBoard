package request

import (
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateFieldRequestToParams(req *openapi.CreateFieldRequest) (name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int, err error) {
	fieldType = entity.FieldType(req.FieldType)
	if !fieldType.IsValid() {
		return "", "", "", false, nil, 0, fmt.Errorf("invalid field_type: %q", req.FieldType)
	}
	entityType = entity.EntityType(req.EntityType)
	if !entityType.IsValid() {
		return "", "", "", false, nil, 0, fmt.Errorf("invalid entity_type: %q", req.EntityType)
	}
	return req.Name, fieldType, entityType,
		derefOr(req.Required, false),
		derefOr(req.Options, nil),
		derefOr(req.OrderIndex, 0),
		nil
}

func UpdateFieldRequestToParams(req *openapi.UpdateFieldRequest) (name string, fieldType entity.FieldType, required bool, options []string, orderIndex int, err error) {
	fieldType = entity.FieldType(req.FieldType)
	if !fieldType.IsValid() {
		return "", "", false, nil, 0, fmt.Errorf("invalid field_type: %q", req.FieldType)
	}
	return req.Name, fieldType,
		derefOr(req.Required, false),
		derefOr(req.Options, nil),
		derefOr(req.OrderIndex, 0),
		nil
}
