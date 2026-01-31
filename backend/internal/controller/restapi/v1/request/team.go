package request

type CreateTeamRequest struct {
	Name         string `json:"name" validate:"required,team_name" example:"Team A"`
	ConfirmReset bool   `json:"confirm_reset"`
}

type JoinTeamRequest struct {
	InviteToken  string `json:"invite_token" validate:"required,not_empty" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	ConfirmReset bool   `json:"confirm_reset"`
}

type TransferCaptainRequest struct {
	NewCaptainId string `json:"new_captain_id" validate:"required,uuid" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
}
