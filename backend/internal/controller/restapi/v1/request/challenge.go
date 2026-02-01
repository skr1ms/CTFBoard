package request

type CreateChallengeRequest struct {
	Title             string  `json:"title" Validate:"required,challenge_title" example:"Challenge 1"`
	Description       string  `json:"description" Validate:"required,challenge_description" example:"Challenge description"`
	Category          string  `json:"category" Validate:"required,challenge_category" example:"Web"`
	Points            int     `json:"points" Validate:"required,gte=0" example:"100"`
	InitialValue      int     `json:"initial_value" Validate:"gte=0" example:"500"`
	MinValue          int     `json:"min_value" Validate:"gte=0" example:"100"`
	Decay             int     `json:"decay" Validate:"gte=0" example:"20"`
	Flag              string  `json:"flag" Validate:"required,challenge_flag" example:"CTF{flag_here}"`
	IsHidden          bool    `json:"is_hidden" example:"false"`
	IsRegex           bool    `json:"is_regex" example:"false"`
	IsCaseInsensitive bool    `json:"is_case_insensitive" example:"false"`
	FlagFormatRegex   *string `json:"flag_format_regex,omitempty"`
}

type SubmitFlagRequest struct {
	Flag string `json:"flag" Validate:"required,not_empty" example:"CTF{flag_here}"`
}

type UpdateChallengeRequest struct {
	Title             string  `json:"title" Validate:"required,challenge_title" example:"Updated title"`
	Description       string  `json:"description" Validate:"required,challenge_description" example:"Updated description"`
	Category          string  `json:"category" Validate:"required,challenge_category" example:"Web"`
	Points            int     `json:"points" Validate:"required,gte=0" example:"150"`
	InitialValue      int     `json:"initial_value" Validate:"gte=0" example:"500"`
	MinValue          int     `json:"min_value" Validate:"gte=0" example:"100"`
	Decay             int     `json:"decay" Validate:"gte=0" example:"20"`
	Flag              string  `json:"flag" Validate:"omitempty,challenge_flag" example:"CTF{new_flag}"`
	IsHidden          bool    `json:"is_hidden" example:"false"`
	IsRegex           bool    `json:"is_regex" example:"false"`
	IsCaseInsensitive bool    `json:"is_case_insensitive" example:"false"`
	FlagFormatRegex   *string `json:"flag_format_regex,omitempty"`
}
