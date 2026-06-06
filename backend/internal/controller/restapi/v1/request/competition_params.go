package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func SetConfigRequestToValueType(v *openapi.SetConfigRequestValueType) (domain.CompetitionParamValueType, error) {
	if v == nil {
		return domain.CompetitionParamTypeString, nil
	}

	switch *v {
	case openapi.SetConfigRequestValueTypeInt:
		return domain.CompetitionParamTypeInt, nil
	case openapi.SetConfigRequestValueTypeBool:
		return domain.CompetitionParamTypeBool, nil
	case openapi.SetConfigRequestValueTypeJSON:
		return domain.CompetitionParamTypeJSON, nil
	case openapi.SetConfigRequestValueTypeString:
		return domain.CompetitionParamTypeString, nil
	default:
		return domain.CompetitionParamTypeString, apperr.NewValidationErrorf("invalid value_type")
	}
}

func SetConfigRequestToParams(req *openapi.SetConfigRequest) (usecase.CompetitionParamSetParams, error) {
	valueType, err := SetConfigRequestToValueType(req.ValueType)
	if err != nil {
		return usecase.CompetitionParamSetParams{}, err
	}

	return usecase.CompetitionParamSetParams{
		Value:       req.Value,
		Description: lo.FromPtrOr(req.Description, ""),
		ValueType:   valueType,
		Category:    lo.FromPtrOr(req.Category, ""),
	}, nil
}

func batchSetConfigItemValueType(v *openapi.BatchSetConfigItemValueType) (domain.CompetitionParamValueType, error) {
	if v == nil {
		return domain.CompetitionParamTypeString, nil
	}

	switch *v {
	case openapi.BatchSetConfigItemValueTypeInt:
		return domain.CompetitionParamTypeInt, nil
	case openapi.BatchSetConfigItemValueTypeBool:
		return domain.CompetitionParamTypeBool, nil
	case openapi.BatchSetConfigItemValueTypeJSON:
		return domain.CompetitionParamTypeJSON, nil
	case openapi.BatchSetConfigItemValueTypeString:
		return domain.CompetitionParamTypeString, nil
	default:
		return domain.CompetitionParamTypeString, apperr.NewValidationErrorf("invalid value_type")
	}
}

func BatchSetConfigRequestToParams(req *openapi.BatchSetConfigRequest) ([]*domain.CompetitionParam, error) {
	if req == nil {
		return nil, nil
	}

	out := make([]*domain.CompetitionParam, 0, len(req.Configs))
	for i := range req.Configs {
		item := &req.Configs[i]

		valueType, err := batchSetConfigItemValueType(item.ValueType)
		if err != nil {
			return nil, err
		}

		out = append(out, &domain.CompetitionParam{
			Key:         item.Key,
			Value:       item.Value,
			ValueType:   valueType,
			Category:    lo.FromPtrOr(item.Category, ""),
			Description: lo.FromPtrOr(item.Description, ""),
		})
	}

	return out, nil
}
