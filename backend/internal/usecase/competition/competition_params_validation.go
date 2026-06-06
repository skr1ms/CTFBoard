package competition

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func validateCategory(category string) error {
	if category == "" {
		return apperr.NewValidationErrorf("category is required")
	}

	if _, ok := allowedCategories[category]; !ok {
		return apperr.NewValidationErrorf("invalid category %q: must be one of general, theme, visibility, scoring, email, social, legal, advanced", category)
	}

	return nil
}

func validateRegisteredParamValue(key, value string) error {
	allowed, ok := allowedVisibilityValues[key]
	if !ok {
		return nil
	}

	if slices.Contains(allowed, value) {
		return nil
	}

	return apperr.NewValidationErrorf("invalid %s %q: must be one of %s", key, value, strings.Join(allowed, ", "))
}

func validateCompetitionParamKey(key string) error {
	if key == "" {
		return apperr.ErrCompetitionParamKeyRequired
	}

	if len(key) > competitionParamKeyMaxLen {
		return apperr.NewValidationErrorf("config key must be at most %d characters", competitionParamKeyMaxLen)
	}

	if !competitionParamKeyRe.MatchString(key) {
		return apperr.NewValidationErrorf("config key must contain only letters, digits, dots, underscores and hyphens")
	}

	return nil
}

// validateValueType checks that value is parseable as the declared type:
// int -> strconv.Atoi, bool -> "true"/"false", string -> always valid,
// json -> json.Valid. Any unknown type also returns ErrCompetitionParamInvalidValueType.
func (uc *CompetitionParamUseCase) validateValueType(valueType domain.CompetitionParamValueType, value string) error {
	switch valueType {
	case domain.CompetitionParamTypeInt:
		if _, err := strconv.Atoi(value); err != nil {
			return apperr.ErrCompetitionParamInvalidValueType
		}
	case domain.CompetitionParamTypeBool:
		if value != "true" && value != "false" {
			return apperr.ErrCompetitionParamInvalidValueType
		}
	case domain.CompetitionParamTypeString:
	case domain.CompetitionParamTypeJSON:
		if !json.Valid([]byte(value)) {
			return apperr.ErrCompetitionParamInvalidValueType
		}
	default:
		return apperr.ErrCompetitionParamInvalidValueType
	}

	return nil
}
