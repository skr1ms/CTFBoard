package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromField(f *domain.Field) openapi.FieldResponse {
	var opts *[]string
	if len(f.Options) > 0 {
		opts = &f.Options
	}
	var fieldType openapi.FieldResponseFieldType
	switch f.FieldType {
	case domain.FieldTypeText:
		fieldType = openapi.FieldResponseFieldTypeText
	case domain.FieldTypeNumber:
		fieldType = openapi.FieldResponseFieldTypeNumber
	case domain.FieldTypeSelect:
		fieldType = openapi.FieldResponseFieldTypeSelect
	case domain.FieldTypeBoolean:
		fieldType = openapi.FieldResponseFieldTypeBoolean
	default:
		fieldType = openapi.FieldResponseFieldTypeText
	}
	var entityType openapi.FieldResponseEntityType
	switch f.EntityType {
	case domain.EntityTypeUser:
		entityType = openapi.FieldResponseEntityTypeUser
	case domain.EntityTypeTeam:
		entityType = openapi.FieldResponseEntityTypeTeam
	default:
		entityType = openapi.FieldResponseEntityTypeUser
	}
	return openapi.FieldResponse{
		ID:         httputil.Ptr(f.ID.String()),
		Name:       httputil.Ptr(f.Name),
		FieldType:  httputil.Ptr(fieldType),
		EntityType: httputil.Ptr(entityType),
		Required:   httputil.Ptr(f.Required),
		Options:    opts,
		OrderIndex: httputil.Ptr(f.OrderIndex),
		CreatedAt:  httputil.Ptr(f.CreatedAt),
	}
}

func FromFieldList(items []*domain.Field) []openapi.FieldResponse {
	return lo.Map(items, func(item *domain.Field, _ int) openapi.FieldResponse { return FromField(item) })
}
