package request

import "github.com/skr1ms/CTFBoard/internal/openapi"

func LoginRequestCredentials(req *openapi.RequestLoginRequest) (email, password string) {
	if req.Email != nil {
		email = *req.Email
	}
	return email, req.Password
}

func RegisterRequestCredentials(req *openapi.RequestRegisterRequest) (username, email, password string) {
	if req.Username != nil {
		username = *req.Username
	}
	if req.Email != nil {
		email = *req.Email
	}
	if req.Password != nil {
		password = *req.Password
	}
	return username, email, password
}
