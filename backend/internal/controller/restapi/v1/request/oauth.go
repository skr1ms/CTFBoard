package request

import (
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func OAuthExchangeRequestToParams(req *openapi.OAuthExchangeRequest) (string, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return "", apperr.NewValidationErrorf("code is required")
	}

	return code, nil
}
