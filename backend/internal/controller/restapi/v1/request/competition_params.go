package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func SetConfigRequestToValueType(v *openapi.SetConfigRequestValueType) entity.CompetitionParamValueType {
	if v == nil {
		return entity.CompetitionParamTypeString
	}
	switch *v {
	case openapi.Int:
		return entity.CompetitionParamTypeInt
	case openapi.Bool:
		return entity.CompetitionParamTypeBool
	case openapi.JSON:
		return entity.CompetitionParamTypeJSON
	case openapi.String:
		return entity.CompetitionParamTypeString
	default:
		return entity.CompetitionParamTypeString
	}
}

type SetConfigParams struct {
	Value       string
	Description string
	ValueType   entity.CompetitionParamValueType
}

func SetConfigRequestToParams(req *openapi.SetConfigRequest) SetConfigParams {
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	return SetConfigParams{
		Value:       req.Value,
		Description: description,
		ValueType:   SetConfigRequestToValueType(req.ValueType),
	}
}
