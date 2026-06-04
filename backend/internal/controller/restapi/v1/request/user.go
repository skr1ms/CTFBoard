package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func BanUserRequestToParams(req *openapi.BanUserRequest) string {
	return req.Reason
}

func AdminCreateUserRequestToParams(req *openapi.AdminCreateUserRequest) (username, email, password, role string, err error) {
	role = ""

	if req.Role != nil {
		r := *req.Role
		if r != "user" && r != "admin" {
			return "", "", "", "", apperr.NewValidationErrorf("role must be 'user' or 'admin'")
		}

		role = r
	}

	return req.Username, req.Email, req.Password, role, nil
}

func AdminUpdateUserRequestToParams(req *openapi.AdminUpdateUserRequest) (username, email, role, password *string, isVerified *bool, err error) {
	if req.Role != nil {
		r := *req.Role
		if r != "user" && r != "admin" {
			return nil, nil, nil, nil, nil, apperr.NewValidationErrorf("role must be 'user' or 'admin'")
		}
	}

	return req.Username, req.Email, req.Role, req.Password, req.IsVerified, nil
}

func UpdateProfileRequestToParams(req *openapi.UpdateProfileRequest) (username, email, currentPassword, newPassword *string) {
	return req.Username, req.Email, req.CurrentPassword, req.Password
}

func LoginRequestToParams(req *openapi.LoginRequest) (email, password string) {
	return req.Email, req.Password
}

func RegisterRequestToParams(req *openapi.RegisterRequest) (username, email, password string, customFields map[string]string, err error) {
	username = req.Username
	email = req.Email
	password = req.Password

	if req.CustomFields != nil {
		customFields = *req.CustomFields
	}

	return username, email, password, customFields, nil
}
