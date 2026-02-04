package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromAPIToken(t *entity.APIToken) openapi.ResponseAPITokenResponse {
	res := openapi.ResponseAPITokenResponse{
		ID:        ptr(t.ID.String()),
		CreatedAt: ptr(t.CreatedAt.Format(time.RFC3339)),
	}
	if t.Description != "" {
		res.Description = ptr(t.Description)
	}
	if t.ExpiresAt != nil {
		res.ExpiresAt = t.ExpiresAt
	}
	if t.LastUsedAt != nil {
		res.LastUsedAt = t.LastUsedAt
	}
	return res
}

func FromAPITokenCreated(plaintext string, t *entity.APIToken) openapi.ResponseAPITokenCreatedResponse {
	res := openapi.ResponseAPITokenCreatedResponse{
		ID:        ptr(t.ID.String()),
		Token:     plaintext,
		CreatedAt: ptr(t.CreatedAt.Format(time.RFC3339)),
	}
	if t.Description != "" {
		res.Description = ptr(t.Description)
	}
	if t.ExpiresAt != nil {
		res.ExpiresAt = t.ExpiresAt
	}
	return res
}
