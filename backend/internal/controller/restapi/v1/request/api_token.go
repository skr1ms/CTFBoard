package request

import (
	"time"

	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const apiTokenDescriptionMaxLength = 255

func CreateAPITokenRequestToParams(req *openapi.CreateAPITokenRequest) (description string, expiresAt *time.Time, err error) {
	expiresAt = req.ExpiresAt
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return "", nil, httperr.NewValidationErrorf("expires_at must be in the future")
	}

	description = lo.FromPtrOr(req.Description, "")
	if len(description) > apiTokenDescriptionMaxLength {
		return "", nil, httperr.NewValidationErrorf("description must be at most %d characters", apiTokenDescriptionMaxLength)
	}

	return description, expiresAt, nil
}
