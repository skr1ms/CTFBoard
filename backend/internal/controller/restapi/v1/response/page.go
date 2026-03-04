package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromPage(p *entity.Page) openapi.PageResponse {
	return openapi.PageResponse{
		ID:         ptr(p.ID.String()),
		Title:      ptr(p.Title),
		Slug:       ptr(p.Slug),
		Content:    ptr(p.Content),
		IsDraft:    ptr(p.IsDraft),
		OrderIndex: ptr(p.OrderIndex),
		CreatedAt:  ptr(p.CreatedAt),
		UpdatedAt:  ptr(p.UpdatedAt),
	}
}

func FromPageListItem(item *entity.PageListItem) openapi.PageListItemResponse {
	return openapi.PageListItemResponse{
		ID:         ptr(item.ID.String()),
		Title:      ptr(item.Title),
		Slug:       ptr(item.Slug),
		OrderIndex: ptr(item.OrderIndex),
	}
}

func FromPageList(items []*entity.PageListItem) []openapi.PageListItemResponse {
	res := make([]openapi.PageListItemResponse, len(items))
	for i, item := range items {
		res[i] = FromPageListItem(item)
	}
	return res
}

func FromPageFullList(items []*entity.Page) []openapi.PageResponse {
	res := make([]openapi.PageResponse, len(items))
	for i, item := range items {
		res[i] = FromPage(item)
	}
	return res
}
