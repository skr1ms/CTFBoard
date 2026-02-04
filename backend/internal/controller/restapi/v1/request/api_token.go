package request

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func CreateAPITokenParams(req *openapi.RequestCreateAPITokenRequest) (description string, expiresAt *time.Time) {
	if req.Description != nil {
		description = *req.Description
	}
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt
	}
	return description, expiresAt
}
