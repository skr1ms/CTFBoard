package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromPage(p *domain.Page) openapi.PageResponse {
	return openapi.PageResponse{
		ID:         new(p.ID.String()),
		Title:      new(p.Title),
		Slug:       new(p.Slug),
		Content:    new(p.Content),
		IsDraft:    new(p.IsDraft),
		OrderIndex: new(p.OrderIndex),
		CreatedAt:  new(p.CreatedAt),
		UpdatedAt:  new(p.UpdatedAt),
	}
}

func FromPageListItem(item *domain.PageListItem) openapi.PageListItemResponse {
	return openapi.PageListItemResponse{
		ID:         new(item.ID.String()),
		Title:      new(item.Title),
		Slug:       new(item.Slug),
		OrderIndex: new(item.OrderIndex),
	}
}

func FromPageList(items []*domain.PageListItem) []openapi.PageListItemResponse {
	return lo.Map(items, func(item *domain.PageListItem, _ int) openapi.PageListItemResponse { return FromPageListItem(item) })
}

func FromPageFullList(items []*domain.Page) []openapi.PageResponse {
	return lo.Map(items, func(item *domain.Page, _ int) openapi.PageResponse { return FromPage(item) })
}
