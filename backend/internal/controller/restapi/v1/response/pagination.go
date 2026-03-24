package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func PaginationMeta(page, perPage int, total int64) *openapi.PaginationMeta {
	m := httputil.NewPaginationMeta(page, perPage, total)
	return &openapi.PaginationMeta{
		Page:       httputil.Ptr(m.Page),
		PerPage:    httputil.Ptr(m.PerPage),
		Total:      httputil.Ptr(m.Total),
		TotalPages: httputil.Ptr(m.TotalPages),
	}
}

func BuildListResponse[D, R any](items []D, convert func(D) R, total int64, page, perPage int) ([]R, *openapi.PaginationMeta) {
	return lo.Map(items, func(item D, _ int) R { return convert(item) }), PaginationMeta(page, perPage, total)
}
