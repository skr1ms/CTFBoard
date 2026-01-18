package request

type CreateChallengeRequest struct {
	Title        string `json:"title" validate:"required,challenge_title" example:"Challenge 1"`
	Description  string `json:"description" validate:"required,challenge_description" example:"Challenge description"`
	Category     string `json:"category" validate:"required,challenge_category" example:"Web"`
	Points       int    `json:"points" validate:"required,gte=0" example:"100"`
	InitialValue int    `json:"initial_value" validate:"gte=0" example:"500"`
	MinValue     int    `json:"min_value" validate:"gte=0" example:"100"`
	Decay        int    `json:"decay" validate:"gte=0" example:"20"`
	Flag         string `json:"flag" validate:"required,challenge_flag" example:"CTF{flag_here}"`
	IsHidden     bool   `json:"is_hidden" example:"false"`
}

type SubmitFlagRequest struct {
	Flag string `json:"flag" validate:"required,not_empty" example:"CTF{flag_here}"`
}

type UpdateChallengeRequest struct {
	Title        string `json:"title" validate:"required,challenge_title" example:"Updated title"`
	Description  string `json:"description" validate:"required,challenge_description" example:"Updated description"`
	Category     string `json:"category" validate:"required,challenge_category" example:"Web"`
	Points       int    `json:"points" validate:"required,gte=0" example:"150"`
	InitialValue int    `json:"initial_value" validate:"gte=0" example:"500"`
	MinValue     int    `json:"min_value" validate:"gte=0" example:"100"`
	Decay        int    `json:"decay" validate:"gte=0" example:"20"`
	Flag         string `json:"flag" validate:"omitempty,challenge_flag" example:"CTF{new_flag}"`
	IsHidden     bool   `json:"is_hidden" example:"false"`
}
