package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
)

func CreateTeamRequestToParams(req *openapi.CreateTeamRequest) (name string, confirmReset bool) {
	confirmReset = false
	if req.ConfirmReset != nil {
		confirmReset = *req.ConfirmReset
	}
	return req.Name, confirmReset
}

func JoinTeamRequestToParams(req *openapi.JoinTeamRequest) (inviteToken string, confirmReset bool) {
	confirmReset = false
	if req.ConfirmReset != nil {
		confirmReset = *req.ConfirmReset
	}
	return req.InviteToken, confirmReset
}

func TransferCaptainRequestToParams(req *openapi.TransferCaptainRequest) (uuid.UUID, error) {
	if req.NewCaptainID == "" {
		return uuid.Nil, helper.NewValidationErrorf("new_captain_id is required")
	}
	parsed, err := uuid.Parse(req.NewCaptainID)
	if err != nil {
		return uuid.Nil, helper.NewValidationErrorf("invalid new_captain_id")
	}
	return parsed, nil
}

func BanTeamRequestToParams(req *openapi.BanTeamRequest) string {
	return req.Reason
}

func SetHiddenRequestToParams(req *openapi.SetHiddenRequest) bool {
	if req.Hidden != nil {
		return *req.Hidden
	}
	return false
}

func AdminUpdateTeamRequestToParams(req *openapi.AdminUpdateTeamRequest) (name *string, captainID, bracketID *uuid.UUID, isHidden *bool, err error) {
	name = req.Name
	isHidden = req.IsHidden

	if req.CaptainID != nil {
		parsed, parseErr := uuid.Parse(*req.CaptainID)
		if parseErr != nil {
			err = helper.NewValidationErrorf("invalid captain_id")
			return name, captainID, bracketID, isHidden, err
		}
		captainID = &parsed
	}

	if req.BracketID != nil {
		parsed, parseErr := uuid.Parse(*req.BracketID)
		if parseErr != nil {
			err = helper.NewValidationErrorf("invalid bracket_id")
			return name, captainID, bracketID, isHidden, err
		}
		bracketID = &parsed
	}

	return name, captainID, bracketID, isHidden, err
}

func UpdateTeamRequestToParams(req *openapi.UpdateTeamRequest) string {
	return req.Name
}

func AdminAddMemberRequestToParams(req *openapi.AdminAddMemberRequest) (uuid.UUID, error) {
	parsed, err := uuid.Parse(req.UserID)
	if err != nil {
		return uuid.Nil, helper.NewValidationErrorf("invalid user_id")
	}
	return parsed, nil
}
