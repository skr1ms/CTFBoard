package request

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func BanUserRequestToParams(req *openapi.BanUserRequest) string {
	return req.Reason
}

func AdminCreateUserRequestToParams(req *openapi.AdminCreateUserRequest) (username, email, password, role string) {
	role = ""
	if req.Role != nil {
		role = *req.Role
	}
	return req.Username, req.Email, req.Password, role
}

func AdminUpdateUserRequestToParams(req *openapi.AdminUpdateUserRequest) (username, email, role, password *string, isVerified *bool) {
	return req.Username, req.Email, req.Role, req.Password, req.IsVerified
}

func UpdateProfileRequestToParams(req *openapi.UpdateProfileRequest) (username, email, currentPassword, newPassword *string) {
	return req.Username, req.Email, req.CurrentPassword, req.Password
}

func LoginRequestToParams(req *openapi.LoginRequest) (email, password string) {
	if req.Email != nil {
		email = *req.Email
	}
	return email, req.Password
}

func RegisterRequestToParams(req *openapi.RegisterRequest) (username, email, password string, customFields map[string]string) {
	if req.Username != nil {
		username = *req.Username
	}
	if req.Email != nil {
		email = *req.Email
	}
	if req.Password != nil {
		password = *req.Password
	}
	if req.CustomFields != nil {
		customFields = *req.CustomFields
	}
	return username, email, password, customFields
}
