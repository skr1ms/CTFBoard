package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type CreateFieldParams = usecase.FieldCreateParams

type UpdateFieldParams = usecase.FieldUpdateParams

func FieldEntityTypeFromParams(entityType *openapi.GetFieldsParamsEntityType) domain.EntityType {
	if entityType != nil && *entityType == openapi.GetFieldsParamsEntityTypeTeam {
		return domain.EntityTypeTeam
	}

	return domain.EntityTypeUser
}

func CreateFieldRequestToParams(req *openapi.CreateFieldRequest) (CreateFieldParams, error) {
	return CreateFieldParams{
		Name:        req.Name,
		Description: lo.FromPtrOr(req.Description, ""),
		FieldType:   domain.FieldType(req.FieldType),
		EntityType:  domain.EntityType(req.EntityType),
		Required:    lo.FromPtrOr(req.Required, false),
		Public:      lo.FromPtrOr(req.Public, false),
		Editable:    lo.FromPtrOr(req.Editable, false),
		Options:     lo.FromPtrOr(req.Options, nil),
		OrderIndex:  lo.FromPtrOr(req.OrderIndex, 0),
	}, nil
}

func UpdateFieldRequestToParams(req *openapi.UpdateFieldRequest) (UpdateFieldParams, error) {
	return UpdateFieldParams{
		Name:        req.Name,
		Description: lo.FromPtrOr(req.Description, ""),
		FieldType:   domain.FieldType(req.FieldType),
		Required:    lo.FromPtrOr(req.Required, false),
		Public:      lo.FromPtrOr(req.Public, false),
		Editable:    lo.FromPtrOr(req.Editable, false),
		Options:     lo.FromPtrOr(req.Options, nil),
		OrderIndex:  lo.FromPtrOr(req.OrderIndex, 0),
	}, nil
}
