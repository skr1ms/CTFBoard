package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromBracket(b *domain.Bracket) openapi.BracketResponse {
	return openapi.BracketResponse{
		ID:          httputil.Ptr(b.ID.String()),
		Name:        httputil.Ptr(b.Name),
		Description: httputil.Ptr(b.Description),
		IsDefault:   httputil.Ptr(b.IsDefault),
		CreatedAt:   httputil.Ptr(b.CreatedAt),
	}
}

func FromBracketList(items []*domain.Bracket) []openapi.BracketResponse {
	return lo.Map(items, func(item *domain.Bracket, _ int) openapi.BracketResponse { return FromBracket(item) })
}
