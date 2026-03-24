package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromPage(p *domain.Page) openapi.PageResponse {
	return openapi.PageResponse{
		ID:         httputil.Ptr(p.ID.String()),
		Title:      httputil.Ptr(p.Title),
		Slug:       httputil.Ptr(p.Slug),
		Content:    httputil.Ptr(p.Content),
		IsDraft:    httputil.Ptr(p.IsDraft),
		OrderIndex: httputil.Ptr(p.OrderIndex),
		CreatedAt:  httputil.Ptr(p.CreatedAt),
		UpdatedAt:  httputil.Ptr(p.UpdatedAt),
	}
}

func FromPageListItem(item *domain.PageListItem) openapi.PageListItemResponse {
	return openapi.PageListItemResponse{
		ID:         httputil.Ptr(item.ID.String()),
		Title:      httputil.Ptr(item.Title),
		Slug:       httputil.Ptr(item.Slug),
		OrderIndex: httputil.Ptr(item.OrderIndex),
	}
}

func FromPageList(items []*domain.PageListItem) []openapi.PageListItemResponse {
	return lo.Map(items, func(item *domain.PageListItem, _ int) openapi.PageListItemResponse { return FromPageListItem(item) })
}

func FromPageFullList(items []*domain.Page) []openapi.PageResponse {
	return lo.Map(items, func(item *domain.Page, _ int) openapi.PageResponse { return FromPage(item) })
}
