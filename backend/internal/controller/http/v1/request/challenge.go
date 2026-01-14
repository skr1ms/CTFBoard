package request

type CreateChallengeRequest struct {
	Title       string `json:"title" validate:"required" example:"Challenge 1"`
	Description string `json:"description" validate:"required" example:"Challenge description"`
	Category    string `json:"category" validate:"required" example:"Web"`
	Points      int    `json:"points" validate:"required" example:"100"`
	Flag        string `json:"flag" validate:"required" example:"CTF{flag_here}"`
	IsHidden    bool   `json:"is_hidden" example:"false"`
}

type SubmitFlagRequest struct {
	Flag string `json:"flag" validate:"required" example:"CTF{flag_here}"`
}

type UpdateChallengeRequest struct {
	Title       string `json:"title" validate:"required" example:"Updated title"`
	Description string `json:"description" validate:"required" example:"Updated description"`
	Category    string `json:"category" validate:"required" example:"Web"`
	Points      int    `json:"points" validate:"required" example:"150"`
	Flag        string `json:"flag" example:"CTF{new_flag}"`
	IsHidden    bool   `json:"is_hidden" example:"false"`
}
