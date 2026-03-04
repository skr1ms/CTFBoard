package request

import (
	"net"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
)

type AdminCreateSubmissionParams struct {
	UserID        uuid.UUID
	TeamID        *uuid.UUID
	ChallengeID   uuid.UUID
	SubmittedFlag string
	IsCorrect     bool
	IP            string
}

func AdminCreateSubmissionRequestToParams(req *openapi.AdminCreateSubmissionRequest) (*AdminCreateSubmissionParams, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, helper.NewValidationErrorf("invalid user_id")
	}

	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return nil, helper.NewValidationErrorf("invalid challenge_id")
	}

	var teamID *uuid.UUID
	if req.TeamID != nil {
		parsed, err := uuid.Parse(*req.TeamID)
		if err != nil {
			return nil, helper.NewValidationErrorf("invalid team_id")
		}
		teamID = &parsed
	}

	ip := ""
	if req.IP != nil {
		if net.ParseIP(*req.IP) == nil {
			return nil, helper.NewValidationErrorf("invalid ip address format")
		}
		ip = *req.IP
	}

	return &AdminCreateSubmissionParams{
		UserID:        userID,
		TeamID:        teamID,
		ChallengeID:   challengeID,
		SubmittedFlag: req.SubmittedFlag,
		IsCorrect:     req.IsCorrect,
		IP:            ip,
	}, nil
}

func AdminUpdateSubmissionRequestToParams(req *openapi.AdminUpdateSubmissionRequest) bool {
	if req.IsCorrect != nil {
		return *req.IsCorrect
	}
	return false
}
