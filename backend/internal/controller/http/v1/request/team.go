package request

type CreateTeamRequest struct {
	Name string `json:"name" validate:"required" example:"Team A"`
}

type JoinTeamRequest struct {
	InviteToken string `json:"invite_token" validate:"required" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
}
