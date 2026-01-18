package request

type RegisterRequest struct {
	Username string `json:"username" validate:"custom_username" example:"player1"`
	Email    string `json:"email" validate:"custom_email" example:"player1@example.com"`
	Password string `json:"password" validate:"strong_password" example:"SecurePassword123!"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"custom_email" example:"player1@example.com"`
	Password string `json:"password" validate:"required" example:"SecurePassword123!"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"custom_email" example:"player1@example.com"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"strong_password" example:"NewSecurePassword123!"`
}
