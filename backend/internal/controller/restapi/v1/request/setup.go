package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// SetupRequestToParams converts the generated transport DTO into the setup use-case request.
func SetupRequestToParams(req *openapi.SetupRequest, clientIP string) *usecase.SetupRequest {
	if req == nil {
		return &usecase.SetupRequest{ClientIP: clientIP}
	}

	return &usecase.SetupRequest{
		CTFName:                   req.CtfName,
		CTFDescription:            stringFromPtr(req.CtfDescription),
		Mode:                      string(req.Mode),
		MaxTeamSize:               intFromPtr(req.MaxTeamSize),
		ChallengeVisibility:       string(req.ChallengeVisibility),
		ScoreVisibility:           string(req.ScoreVisibility),
		AccountVisibility:         string(req.AccountVisibility),
		RegistrationVisibility:    string(req.RegistrationVisibility),
		EmailVerificationRequired: boolFromPtr(req.EmailVerificationRequired),
		AdminUsername:             req.AdminUsername,
		AdminEmail:                req.AdminEmail,
		AdminPassword:             req.AdminPassword,
		StartTime:                 req.StartTime,
		EndTime:                   req.EndTime,
		FreezeTime:                req.FreezeTime,
		Timezone:                  stringFromPtr(req.Timezone),
		ClientIP:                  clientIP,
	}
}

func stringFromPtr(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

func intFromPtr(v *int) int {
	if v == nil {
		return 0
	}

	return *v
}

func boolFromPtr(v *bool) bool {
	if v == nil {
		return false
	}

	return *v
}
