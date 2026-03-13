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
	case openapi.SetConfigRequestValueTypeInt:
		return entity.CompetitionParamTypeInt, nil
	case openapi.SetConfigRequestValueTypeBool:
		return entity.CompetitionParamTypeBool, nil
	case openapi.SetConfigRequestValueTypeJSON:
		return entity.CompetitionParamTypeJSON, nil
	case openapi.SetConfigRequestValueTypeString:
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

func batchSetConfigItemValueType(v *openapi.BatchSetConfigItemValueType) (entity.CompetitionParamValueType, error) {
	if v == nil {
		return entity.CompetitionParamTypeString, nil
	}
	switch *v {
	case openapi.BatchSetConfigItemValueTypeInt:
		return entity.CompetitionParamTypeInt, nil
	case openapi.BatchSetConfigItemValueTypeBool:
		return entity.CompetitionParamTypeBool, nil
	case openapi.BatchSetConfigItemValueTypeJSON:
		return entity.CompetitionParamTypeJSON, nil
	case openapi.BatchSetConfigItemValueTypeString:
		return entity.CompetitionParamTypeString, nil
	default:
		return entity.CompetitionParamTypeString, helper.NewValidationErrorf("invalid value_type")
	}
}

func BatchSetConfigRequestToParams(req *openapi.BatchSetConfigRequest) ([]*entity.CompetitionParam, error) {
	if req == nil {
		return nil, nil
	}
	out := make([]*entity.CompetitionParam, 0, len(req.Configs))
	for i := range req.Configs {
		item := &req.Configs[i]
		if item.Key == "" {
			return nil, helper.NewValidationErrorf("configs[%d]: key required", i)
		}
		desc := ""
		if item.Description != nil {
			desc = *item.Description
		}
		if len(item.Value) > maxConfigValueLength {
			return nil, helper.NewValidationErrorf("configs[%d]: value too long", i)
		}
		if len(desc) > maxConfigDescriptionLength {
			return nil, helper.NewValidationErrorf("configs[%d]: description too long", i)
		}
		valueType, err := batchSetConfigItemValueType(item.ValueType)
		if err != nil {
			return nil, err
		}
		out = append(out, &entity.CompetitionParam{
			Key:         item.Key,
			Value:       item.Value,
			ValueType:   valueType,
			Description: desc,
		})
	}
	return out, nil
}
