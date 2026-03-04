package request

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateAPITokenRequestToParams(req *openapi.CreateAPITokenRequest) (description string, expiresAt *time.Time) {
	return derefOr(req.Description, ""), req.ExpiresAt
}
