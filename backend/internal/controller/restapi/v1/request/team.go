package request

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
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
	parsed, err := uuid.Parse(req.NewCaptainID)
	if err != nil {
		return uuid.Nil, apperr.NewValidationErrorf("invalid new_captain_id")
	}

	return parsed, nil
}

func BanTeamRequestToParams(req *openapi.BanTeamRequest) (reason string, banMembers bool) {
	reason = req.Reason
	banMembers = lo.FromPtrOr(req.BanMembers, false)

	return reason, banMembers
}

func BulkBanTeamsRequestToParams(req *openapi.BulkBanTeamsRequest) (ids []uuid.UUID, reason string, banMembers bool) {
	reason = req.Reason
	banMembers = lo.FromPtrOr(req.BanMembers, false)

	return req.Ids, reason, banMembers
}

func BulkTeamIDsRequestToParams(req *openapi.BulkTeamIDsRequest) []uuid.UUID {
	return req.Ids
}

func BulkSetHiddenRequestToParams(req *openapi.BulkSetHiddenRequest) ([]uuid.UUID, bool) {
	return req.Ids, req.Hidden
}

func SetHiddenRequestToParams(req *openapi.SetHiddenRequest) (*bool, error) {
	v := req.Hidden

	return &v, nil
}

func AdminTeamsBanStatusFromParams(params openapi.GetAdminTeamsParams) (usecase.AdminTeamBanStatus, error) {
	if params.BanStatus == nil {
		return usecase.AdminTeamBanStatusAll, nil
	}

	if !params.BanStatus.Valid() {
		return "", apperr.NewValidationErrorf("ban_status must be one of: all, not_banned, banned")
	}

	return usecase.AdminTeamBanStatus(*params.BanStatus), nil
}

func AdminTeamsVisibilityFromParams(params openapi.GetAdminTeamsParams) (usecase.AdminTeamVisibility, error) {
	if params.Visibility == nil {
		return usecase.AdminTeamVisibilityAll, nil
	}

	if !params.Visibility.Valid() {
		return "", apperr.NewValidationErrorf("visibility must be one of: all, visible, hidden")
	}

	return usecase.AdminTeamVisibility(*params.Visibility), nil
}

func AdminUpdateTeamRequestToParams(req *openapi.AdminUpdateTeamRequest) (name *string, captainID, bracketID *uuid.UUID, isHidden *bool, err error) {
	name = req.Name
	if name != nil && *name == "" {
		err = apperr.NewValidationErrorf("name cannot be empty")

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

func UpdateTeamRequestToParams(req *openapi.UpdateTeamRequest) (usecase.TeamUpdateParams, error) {
	if req.Name == nil && req.CustomFields == nil {
		return usecase.TeamUpdateParams{}, apperr.NewValidationErrorf("at least one team field must be provided")
	}

	if req.Name != nil && *req.Name == "" {
		return usecase.TeamUpdateParams{}, apperr.NewValidationErrorf("name cannot be empty")
	}

	return usecase.TeamUpdateParams{
		Name:         req.Name,
		CustomFields: req.CustomFields,
	}, nil
}

func AdminAddMemberRequestToParams(req *openapi.AdminAddMemberRequest) (uuid.UUID, error) {
	return req.UserID, nil
}
