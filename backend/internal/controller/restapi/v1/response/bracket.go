package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromBracket(b *entity.Bracket) openapi.ResponseBracketResponse {
	return openapi.ResponseBracketResponse{
		ID:          ptr(b.ID.String()),
		Name:        ptr(b.Name),
		Description: ptr(b.Description),
		IsDefault:   ptr(b.IsDefault),
		CreatedAt:   ptr(b.CreatedAt),
	}
}

func FromBracketList(items []*entity.Bracket) []openapi.ResponseBracketResponse {
	res := make([]openapi.ResponseBracketResponse, len(items))
	for i, b := range items {
		res[i] = FromBracket(b)
	}
	return res
}
