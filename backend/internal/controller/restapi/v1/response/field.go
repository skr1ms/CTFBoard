package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromField(f *entity.Field) openapi.FieldResponse {
	var opts *[]string
	if len(f.Options) > 0 {
		opts = &f.Options
	}
	var fieldType openapi.FieldResponseFieldType
	switch f.FieldType {
	case entity.FieldTypeText:
		fieldType = openapi.FieldResponseFieldTypeText
	case entity.FieldTypeNumber:
		fieldType = openapi.FieldResponseFieldTypeNumber
	case entity.FieldTypeSelect:
		fieldType = openapi.FieldResponseFieldTypeSelect
	case entity.FieldTypeBoolean:
		fieldType = openapi.FieldResponseFieldTypeBoolean
	default:
		fieldType = openapi.FieldResponseFieldTypeText
	}
	var entityType openapi.FieldResponseEntityType
	switch f.EntityType {
	case entity.EntityTypeUser:
		entityType = openapi.FieldResponseEntityTypeUser
	case entity.EntityTypeTeam:
		entityType = openapi.FieldResponseEntityTypeTeam
	default:
		entityType = openapi.FieldResponseEntityTypeUser
	}
	return openapi.FieldResponse{
		ID:         ptr(f.ID.String()),
		Name:       ptr(f.Name),
		FieldType:  ptr(fieldType),
		EntityType: ptr(entityType),
		Required:   ptr(f.Required),
		Options:    opts,
		OrderIndex: ptr(f.OrderIndex),
		CreatedAt:  ptr(f.CreatedAt),
	}
}

func FromFieldList(items []*entity.Field) []openapi.FieldResponse {
	res := make([]openapi.FieldResponse, len(items))
	for i, item := range items {
		res[i] = FromField(item)
	}
	return res
}
