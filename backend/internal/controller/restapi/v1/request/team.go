package request

type CreateTeamRequest struct {
	Name         string `json:"name" Validate:"required,team_name" example:"Team A"`
	ConfirmReset bool   `json:"confirm_reset"`
}

type JoinTeamRequest struct {
	InviteToken  string `json:"invite_token" Validate:"required,not_empty" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	ConfirmReset bool   `json:"confirm_reset"`
}

type TransferCaptainRequest struct {
	NewCaptainID string `json:"new_captain_id" Validate:"required,uuid" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
}

type BanTeamRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

type SetHiddenRequest struct {
	Hidden bool `json:"hidden"`
}
