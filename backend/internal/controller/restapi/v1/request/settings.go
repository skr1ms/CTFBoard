package request

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func UpdateAppSettingsRequestToEntity(req *openapi.RequestUpdateAppSettingsRequest, id int) *entity.AppSettings {
	var verifyEmails, resendEnabled, registrationOpen bool
	var verifyTTLHours, resetTTLHours, submitLimitPerUser, submitLimitDurationMin int
	scoreboardVisible := "public"
	if req.VerifyEmails != nil {
		verifyEmails = *req.VerifyEmails
	}
	if req.ResendEnabled != nil {
		resendEnabled = *req.ResendEnabled
	}
	if req.RegistrationOpen != nil {
		registrationOpen = *req.RegistrationOpen
	}
	if req.VerifyTTLHours != nil {
		verifyTTLHours = *req.VerifyTTLHours
	} else {
		verifyTTLHours = 24
	}
	if req.ResetTTLHours != nil {
		resetTTLHours = *req.ResetTTLHours
	} else {
		resetTTLHours = 24
	}
	if req.SubmitLimitPerUser != nil {
		submitLimitPerUser = *req.SubmitLimitPerUser
	} else {
		submitLimitPerUser = 10
	}
	if req.SubmitLimitDurationMin != nil {
		submitLimitDurationMin = *req.SubmitLimitDurationMin
	} else {
		submitLimitDurationMin = 5
	}
	if req.ScoreboardVisible != nil {
		scoreboardVisible = string(*req.ScoreboardVisible)
	}
	return &entity.AppSettings{
		ID:                     id,
		AppName:                req.AppName,
		VerifyEmails:           verifyEmails,
		FrontendURL:            req.FrontendURL,
		CORSOrigins:            req.CorsOrigins,
		ResendEnabled:          resendEnabled,
		ResendFromEmail:        req.ResendFromEmail,
		ResendFromName:         req.ResendFromName,
		VerifyTTLHours:         verifyTTLHours,
		ResetTTLHours:          resetTTLHours,
		SubmitLimitPerUser:     submitLimitPerUser,
		SubmitLimitDurationMin: submitLimitDurationMin,
		ScoreboardVisible:      scoreboardVisible,
		RegistrationOpen:       registrationOpen,
	}
}
