package request

import "github.com/skr1ms/CTFBoard/internal/openapi"

func ForgotPasswordRequestEmail(req *openapi.RequestForgotPasswordRequest) string {
	if req.Email != nil {
		return *req.Email
	}
	return ""
}

func ResetPasswordRequestParams(req *openapi.RequestResetPasswordRequest) (token, newPassword string) {
	if req.NewPassword != nil {
		newPassword = *req.NewPassword
	}
	return req.Token, newPassword
}
