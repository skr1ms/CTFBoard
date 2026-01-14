package entity

type Challenge struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Points      int    `json:"points"`
	FlagHash    string `json:"flag_hash"`
	IsHidden    bool   `json:"is_hidden"`
}
