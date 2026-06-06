package request

import (
	"net"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func AdminCreateSubmissionRequestToParams(req *openapi.AdminCreateSubmissionRequest) (usecase.AdminCreateSubmissionParams, error) {
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
			return usecase.AdminCreateSubmissionParams{}, apperr.NewValidationErrorf("invalid ip address format")
		}

		ip = *req.IP
	}

	return usecase.AdminCreateSubmissionParams{
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
		return nil, apperr.NewValidationErrorf("is_correct is required")
	}

	return &req.IsCorrect, nil
}
