package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func CreateShareRequestToParams(req *openapi.CreateShareRequest, userID, teamID uuid.UUID) (usecase.CreateShareParams, error) {
	shareType := string(req.Type)
	if shareType != usecase.ShareTypeSolve {
		return usecase.CreateShareParams{}, apperr.NewValidationErrorf("share type must be solve")
	}

	return usecase.CreateShareParams{
		Type:        shareType,
		UserID:      userID,
		TeamID:      teamID,
		ChallengeID: uuid.UUID(req.ChallengeID),
	}, nil
}
