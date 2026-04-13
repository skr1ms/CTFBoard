package domain

import "time"

// CompetitionParamValueType is a string-backed enum for the expected Go type of a dynamic parameter value.
type CompetitionParamValueType string

const (
	// CompetitionParamTypeString indicates the parameter value is a plain string.
	CompetitionParamTypeString CompetitionParamValueType = "string"
	// CompetitionParamTypeInt indicates the parameter value should be parsed as an integer.
	CompetitionParamTypeInt CompetitionParamValueType = "int"
	// CompetitionParamTypeBool indicates the parameter value should be parsed as a boolean.
	CompetitionParamTypeBool CompetitionParamValueType = "bool"
	// CompetitionParamTypeJSON indicates the parameter value is a JSON-encoded object or array.
	CompetitionParamTypeJSON CompetitionParamValueType = "json"
)

// CompetitionParam is a dynamic key-value configuration entry stored in the database
// and associated with a competition.
type CompetitionParam struct {
	Key         string                    `json:"key"`
	Value       string                    `json:"value"`
	ValueType   CompetitionParamValueType `json:"value_type"`
	Description string                    `json:"description,omitempty"`
	Category    string                    `json:"category"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}
