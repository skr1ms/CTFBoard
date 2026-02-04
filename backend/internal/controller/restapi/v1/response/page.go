package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromPage(p *entity.Page) openapi.ResponsePageResponse {
	return openapi.ResponsePageResponse{
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

func FromPageListItem(item *entity.PageListItem) openapi.ResponsePageListItemResponse {
	return openapi.ResponsePageListItemResponse{
		ID:         ptr(item.ID.String()),
		Title:      ptr(item.Title),
		Slug:       ptr(item.Slug),
		OrderIndex: ptr(item.OrderIndex),
	}
}

func FromPageList(items []*entity.PageListItem) []openapi.ResponsePageListItemResponse {
	res := make([]openapi.ResponsePageListItemResponse, len(items))
	for i, item := range items {
		res[i] = FromPageListItem(item)
	}
	return res
}

func FromPageFullList(items []*entity.Page) []openapi.ResponsePageResponse {
	res := make([]openapi.ResponsePageResponse, len(items))
	for i, item := range items {
		res[i] = FromPage(item)
	}
	return res
}
