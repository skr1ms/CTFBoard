package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromField(f *entity.Field) openapi.ResponseFieldResponse {
	var opts *[]string
	if len(f.Options) > 0 {
		opts = &f.Options
	}
	var fieldType openapi.ResponseFieldResponseFieldType
	switch f.FieldType {
	case entity.FieldTypeText:
		fieldType = openapi.Text
	case entity.FieldTypeNumber:
		fieldType = openapi.Number
	case entity.FieldTypeSelect:
		fieldType = openapi.Select
	case entity.FieldTypeBoolean:
		fieldType = openapi.Boolean
	default:
		fieldType = openapi.Text
	}
	var entityType openapi.ResponseFieldResponseEntityType
	switch f.EntityType {
	case entity.EntityTypeUser:
		entityType = openapi.ResponseFieldResponseEntityTypeUser
	case entity.EntityTypeTeam:
		entityType = openapi.ResponseFieldResponseEntityTypeTeam
	default:
		entityType = openapi.ResponseFieldResponseEntityTypeUser
	}
	return openapi.ResponseFieldResponse{
		ID:         ptr(f.ID.String()),
		Name:       ptr(f.Name),
		FieldType:  &fieldType,
		EntityType: &entityType,
		Required:   ptr(f.Required),
		Options:    opts,
		OrderIndex: ptr(f.OrderIndex),
		CreatedAt:  ptr(f.CreatedAt),
	}
}

func FromFieldList(items []*entity.Field) []openapi.ResponseFieldResponse {
	res := make([]openapi.ResponseFieldResponse, len(items))
	for i, item := range items {
		res[i] = FromField(item)
	}
	return res
}
