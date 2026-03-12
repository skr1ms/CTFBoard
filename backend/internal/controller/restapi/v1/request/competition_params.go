package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxConfigValueLength       = 10000
	maxConfigDescriptionLength = 500
)

func SetConfigRequestToValueType(v *openapi.SetConfigRequestValueType) (entity.CompetitionParamValueType, error) {
	if v == nil {
		return entity.CompetitionParamTypeString, nil
	}
	switch *v {
	case openapi.Int:
		return entity.CompetitionParamTypeInt, nil
	case openapi.Bool:
		return entity.CompetitionParamTypeBool, nil
	case openapi.JSON:
		return entity.CompetitionParamTypeJSON, nil
	case openapi.String:
		return entity.CompetitionParamTypeString, nil
	default:
		return entity.CompetitionParamTypeString, helper.NewValidationErrorf("invalid value_type")
	}
}

type SetConfigParams struct {
	Value       string
	Description string
	ValueType   entity.CompetitionParamValueType
}

func SetConfigRequestToParams(req *openapi.SetConfigRequest) (SetConfigParams, error) {
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	if len(req.Value) > maxConfigValueLength {
		return SetConfigParams{}, helper.NewValidationErrorf("value too long")
	}
	if len(description) > maxConfigDescriptionLength {
		return SetConfigParams{}, helper.NewValidationErrorf("description too long")
	}
	valueType, err := SetConfigRequestToValueType(req.ValueType)
	if err != nil {
		return SetConfigParams{}, err
	}
	return SetConfigParams{
		Value:       req.Value,
		Description: description,
		ValueType:   valueType,
	}, nil
}
