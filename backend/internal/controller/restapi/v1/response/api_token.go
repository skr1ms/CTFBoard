package response

import (
	"time"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAPIToken(t *domain.APIToken) openapi.APITokenResponse {
	res := openapi.APITokenResponse{
		ID:        httputil.Ptr(t.ID.String()),
		CreatedAt: httputil.Ptr(t.CreatedAt.Format(time.RFC3339)),
	}
	if t.Description != "" {
		res.Description = httputil.Ptr(t.Description)
	}
	if t.ExpiresAt != nil {
		res.ExpiresAt = t.ExpiresAt
	}
	if t.LastUsedAt != nil {
		res.LastUsedAt = t.LastUsedAt
	}
	return res
}

func FromAPITokenList(ts []*domain.APIToken) []openapi.APITokenResponse {
	return lo.Map(ts, func(t *domain.APIToken, _ int) openapi.APITokenResponse { return FromAPIToken(t) })
}

func FromAPITokenCreated(plaintext string, t *domain.APIToken) openapi.APITokenCreatedResponse {
	res := openapi.APITokenCreatedResponse{
		ID:        httputil.Ptr(t.ID.String()),
		Token:     plaintext,
		CreatedAt: httputil.Ptr(t.CreatedAt.Format(time.RFC3339)),
	}
	if t.Description != "" {
		res.Description = httputil.Ptr(t.Description)
	}
	if t.ExpiresAt != nil {
		res.ExpiresAt = t.ExpiresAt
	}
	return res
}
