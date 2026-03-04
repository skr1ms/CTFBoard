package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromBracket(b *entity.Bracket) openapi.BracketResponse {
	return openapi.BracketResponse{
		ID:          ptr(b.ID.String()),
		Name:        ptr(b.Name),
		Description: ptr(b.Description),
		IsDefault:   ptr(b.IsDefault),
		CreatedAt:   ptr(b.CreatedAt),
	}
}

func FromBracketList(items []*entity.Bracket) []openapi.BracketResponse {
	res := make([]openapi.BracketResponse, len(items))
	for i, b := range items {
		res[i] = FromBracket(b)
	}
	return res
}
