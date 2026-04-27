package request

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func ForgotPasswordRequestToParams(req *openapi.ForgotPasswordRequest) string {
	return req.Email
}

func ResendVerificationByEmailRequestToParams(req *openapi.ResendVerificationByEmailRequest) string {
	return req.Email
}

func ResetPasswordRequestToParams(req *openapi.ResetPasswordRequest) (token, newPassword string) {
	return req.Token, req.NewPassword
}
