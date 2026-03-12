package request

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const apiTokenDescriptionMaxLength = 255

func CreateAPITokenRequestToParams(req *openapi.CreateAPITokenRequest) (description string, expiresAt *time.Time, err error) {
	expiresAt = req.ExpiresAt
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return "", nil, helper.NewValidationErrorf("expires_at must be in the future")
	}
	description = derefOr(req.Description, "")
	if len(description) > apiTokenDescriptionMaxLength {
		return "", nil, helper.NewValidationErrorf("description must be at most %d characters", apiTokenDescriptionMaxLength)
	}
	return description, expiresAt, nil
}
