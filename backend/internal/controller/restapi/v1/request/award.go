package request

type CreateAwardRequest struct {
	TeamID      string `json:"team_id" Validate:"required,uuid"`
	Value       int    `json:"value" Validate:"required,ne=0"`
	Description string `json:"description" Validate:"required"`
}
