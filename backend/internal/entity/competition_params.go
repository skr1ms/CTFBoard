package entity

import "time"

type CompetitionParamValueType string

const (
	CompetitionParamTypeString CompetitionParamValueType = "string"
	CompetitionParamTypeInt    CompetitionParamValueType = "int"
	CompetitionParamTypeBool   CompetitionParamValueType = "bool"
	CompetitionParamTypeJSON   CompetitionParamValueType = "json"
)

type CompetitionParam struct {
	Key         string                    `json:"key"`
	Value       string                    `json:"value"`
	ValueType   CompetitionParamValueType `json:"value_type"`
	Description string                    `json:"description,omitempty"`
	Category    string                    `json:"category"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}
