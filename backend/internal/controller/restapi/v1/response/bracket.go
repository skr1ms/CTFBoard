package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromBracket(b *domain.Bracket) openapi.BracketResponse {
	return openapi.BracketResponse{
		ID:          new(b.ID.String()),
		Name:        new(b.Name),
		Description: new(b.Description),
		IsDefault:   new(b.IsDefault),
		CreatedAt:   new(b.CreatedAt),
	}
}

func FromBracketList(items []*domain.Bracket) []openapi.BracketResponse {
	return lo.Map(items, func(item *domain.Bracket, _ int) openapi.BracketResponse { return FromBracket(item) })
}
