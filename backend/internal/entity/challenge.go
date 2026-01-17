package entity

type Challenge struct {
	Id           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Points       int    `json:"points"`
	InitialValue int    `json:"initial_value"`
	MinValue     int    `json:"min_value"`
	Decay        int    `json:"decay"`
	SolveCount   int    `json:"solve_count"`
	FlagHash     string `json:"flag_hash"`
	IsHidden     bool   `json:"is_hidden"`
}
