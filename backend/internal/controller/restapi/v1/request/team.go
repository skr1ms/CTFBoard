package request

type CreateTeamRequest struct {
	Name string `json:"name" validate:"required,team_name" example:"Team A"`
}

type JoinTeamRequest struct {
	InviteToken string `json:"invite_token" validate:"required,not_empty" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
}
