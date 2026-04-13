package apperr

import "errors"

var (
	ErrCompetitionParamNotFound         = errors.New("competition param not found")
	ErrCompetitionParamKeyRequired      = errors.New("competition param key is required")
	ErrCompetitionParamInvalidValueType = errors.New("invalid value type or value does not match type")
)
