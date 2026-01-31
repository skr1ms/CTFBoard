package request

type RegisterRequest struct {
	Username string `json:"username" Validate:"custom_username" example:"player1"`
	Email    string `json:"email" Validate:"custom_email" example:"player1@example.com"`
	Password string `json:"password" Validate:"strong_password" example:"SecurePassword123!"`
}

type LoginRequest struct {
	Email    string `json:"email" Validate:"custom_email" example:"player1@example.com"`
	Password string `json:"password" Validate:"required" example:"SecurePassword123!"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" Validate:"custom_email" example:"player1@example.com"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" Validate:"required"`
	NewPassword string `json:"new_password" Validate:"strong_password" example:"NewSecurePassword123!"`
}
