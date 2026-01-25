package request

type CreateAwardRequest struct {
	TeamId      string `json:"team_id" validate:"required,uuid"`
	Value       int    `json:"value" validate:"required,ne=0"`
	Description string `json:"description" validate:"required"`
}
