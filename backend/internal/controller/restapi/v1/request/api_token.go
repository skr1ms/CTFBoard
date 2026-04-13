package request

import (
	"time"

	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const apiTokenDescriptionMaxLength = 255

func CreateAPITokenRequestToParams(req *openapi.CreateAPITokenRequest) (description string, expiresAt *time.Time, err error) {
	expiresAt = req.ExpiresAt
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return "", nil, apperr.NewValidationErrorf("expires_at must be in the future")
	}

	description = lo.FromPtrOr(req.Description, "")
	if len(description) > apiTokenDescriptionMaxLength {
		return "", nil, apperr.NewValidationErrorf("description must be at most %d characters", apiTokenDescriptionMaxLength)
	}

	return description, expiresAt, nil
}
