package request

import (
	"net"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	UserSearchFieldUsername = "username"
	UserSearchFieldIP       = "ip"
)

func BanUserRequestToParams(req *openapi.BanUserRequest) string {
	return req.Reason
}

func AdminUsersFieldFromParams(params openapi.GetAdminUsersParams) (string, error) {
	if params.Field == nil {
		return UserSearchFieldUsername, nil
	}

	if !params.Field.Valid() {
		return "", apperr.NewValidationErrorf("field must be one of: username, ip")
	}

	return string(*params.Field), nil
}

func ValidateAdminUsersSearch(field string, q *string) error {
	if field == UserSearchFieldIP && q != nil && net.ParseIP(*q) == nil {
		return apperr.NewValidationErrorf("invalid IP address")
	}

	return nil
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

func UpdateProfileRequestToParams(userID uuid.UUID, req *openapi.UpdateProfileRequest) usecase.UserProfileUpdateParams {
	return usecase.UserProfileUpdateParams{
		UserID:          userID,
		Username:        req.Username,
		Email:           req.Email,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.Password,
		CustomFields:    req.CustomFields,
	}
}

func LoginRequestToParams(req *openapi.LoginRequest) (email, password string) {
	return req.Email, req.Password
}

func RegisterRequestToParams(req *openapi.RegisterRequest) (usecase.UserRegisterParams, error) {
	params := usecase.UserRegisterParams{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	if req.RegistrationCode != nil {
		params.RegistrationCode = *req.RegistrationCode
	}

	if req.CustomFields != nil {
		params.CustomFields = *req.CustomFields
	}

	return params, nil
}
