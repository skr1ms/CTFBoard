package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const maxBatchConfigItems = 50

type setConfigConstraints struct {
	Value       string `validate:"max=10000"`
	Description string `validate:"max=500"`
}

type batchSetConfigItemConstraints struct {
	Key         string `validate:"required"`
	Value       string `validate:"max=10000"`
	Description string `validate:"max=500"`
}

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
		return domain.CompetitionParamTypeString, httperr.NewValidationErrorf("invalid value_type")
	}
}

type SetConfigParams struct {
	Value       string
	Description string
	ValueType   domain.CompetitionParamValueType
	Category    string
}

func ValidateSetConfigRequest(req *openapi.SetConfigRequest, v validator.Validator) error {
	c := setConfigConstraints{Value: req.Value, Description: lo.FromPtrOr(req.Description, "")}

	return ValidateConstraints(v, &c)
}

func SetConfigRequestToParams(req *openapi.SetConfigRequest) (SetConfigParams, error) {
	valueType, err := SetConfigRequestToValueType(req.ValueType)
	if err != nil {
		return SetConfigParams{}, err
	}

	return SetConfigParams{
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
		return domain.CompetitionParamTypeString, httperr.NewValidationErrorf("invalid value_type")
	}
}

func ValidateBatchSetConfigRequest(req *openapi.BatchSetConfigRequest, v validator.Validator) error {
	if req == nil || len(req.Configs) == 0 {
		return nil
	}

	if len(req.Configs) > maxBatchConfigItems {
		return httperr.NewValidationErrorf("configs: at most %d items allowed", maxBatchConfigItems)
	}

	for i := range req.Configs {
		item := &req.Configs[i]

		c := batchSetConfigItemConstraints{
			Key: item.Key, Value: item.Value,
			Description: lo.FromPtrOr(item.Description, ""),
		}

		err := ValidateConstraints(v, &c)
		if err != nil {
			return httperr.NewValidationErrorf("configs[%d]: %v", i, err)
		}
	}

	return nil
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
