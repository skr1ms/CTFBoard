package request

import (
	"net"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type adminCreateSubmissionConstraints struct {
	SubmittedFlag string `validate:"required,max=200"`
}

func ValidateAdminCreateSubmissionRequest(req *openapi.AdminCreateSubmissionRequest, v validator.Validator) error {
	c := adminCreateSubmissionConstraints{SubmittedFlag: req.SubmittedFlag}

	return ValidateConstraints(v, &c)
}

type AdminCreateSubmissionParams struct {
	UserID        uuid.UUID
	TeamID        *uuid.UUID
	ChallengeID   uuid.UUID
	SubmittedFlag string
	IsCorrect     bool
	IP            string
}

func AdminCreateSubmissionRequestToParams(req *openapi.AdminCreateSubmissionRequest) (*AdminCreateSubmissionParams, error) {
	userID := req.UserID
	challengeID := req.ChallengeID

	var teamID *uuid.UUID

	if req.TeamID != nil {
		t := *req.TeamID
		teamID = &t
	}

	ip := ""

	if req.IP != nil {
		if net.ParseIP(*req.IP) == nil {
			return nil, httperr.NewValidationErrorf("invalid ip address format")
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

func AdminUpdateSubmissionRequestToParams(req *openapi.AdminUpdateSubmissionRequest) (*bool, error) {
	if req == nil {
		return nil, httperr.NewValidationErrorf("is_correct is required")
	}

	return &req.IsCorrect, nil
}
