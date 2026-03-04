package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAPIToken(t *entity.APIToken) openapi.APITokenResponse {
	res := openapi.APITokenResponse{
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

func FromAPITokenList(ts []*entity.APIToken) []openapi.APITokenResponse {
	res := make([]openapi.APITokenResponse, len(ts))
	for i, t := range ts {
		res[i] = FromAPIToken(t)
	}
	return res
}

func FromAPITokenCreated(plaintext string, t *entity.APIToken) openapi.APITokenCreatedResponse {
	res := openapi.APITokenCreatedResponse{
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
