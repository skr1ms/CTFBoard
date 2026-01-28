package entity

import "github.com/google/uuid"

type Challenge struct {
	Id                uuid.UUID `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	Points            int       `json:"points"`
	InitialValue      int       `json:"initial_value"`
	MinValue          int       `json:"min_value"`
	Decay             int       `json:"decay"`
	SolveCount        int       `json:"solve_count"`
	FlagHash          string    `json:"flag_hash"`
	IsHidden          bool      `json:"is_hidden"`
	IsRegex           bool      `json:"is_regex"`
	IsCaseInsensitive bool      `json:"is_case_insensitive"`
	FlagRegex         string    `json:"flag_regex,omitempty"`
}
