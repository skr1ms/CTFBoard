package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateTeamRequestToParams(req *openapi.CreateTeamRequest) (name string, confirmReset bool) {
	confirmReset = false
	if req.ConfirmReset != nil {
		confirmReset = *req.ConfirmReset
	}
	return req.Name, confirmReset
}

func CreateSoloTeamRequestToParams(req *openapi.CreateSoloTeamRequest) bool {
	if req.ConfirmReset != nil {
		return *req.ConfirmReset
	}
	return false
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

func BanTeamRequestToParams(req *openapi.BanTeamRequest) (reason string, banMembers bool) {
	reason = req.Reason
	banMembers = derefOr(req.BanMembers, false)
	return reason, banMembers
}

func SetHiddenRequestToParams(req *openapi.SetHiddenRequest) (*bool, error) {
	v := req.Hidden
	return &v, nil
}

func AdminUpdateTeamRequestToParams(req *openapi.AdminUpdateTeamRequest) (name *string, captainID, bracketID *uuid.UUID, isHidden *bool, err error) {
	name = req.Name
	if name != nil && *name == "" {
		err = helper.NewValidationErrorf("name cannot be empty")
		return name, captainID, bracketID, isHidden, err
	}
	isHidden = req.IsHidden

	if req.CaptainID != nil {
		c := *req.CaptainID
		captainID = &c
	}

	if req.BracketID != nil {
		b := *req.BracketID
		bracketID = &b
	}

	return name, captainID, bracketID, isHidden, err
}

func UpdateTeamRequestToParams(req *openapi.UpdateTeamRequest) string {
	return req.Name
}

func AdminAddMemberRequestToParams(req *openapi.AdminAddMemberRequest) (uuid.UUID, error) {
	return req.UserID, nil
}
