package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func PaginationMeta(page, perPage int, total int64) *openapi.PaginationMeta {
	m := httputil.NewPaginationMeta(page, perPage, total)

	return &openapi.PaginationMeta{
		Page:       new(m.Page),
		PerPage:    new(m.PerPage),
		Total:      new(m.Total),
		TotalPages: new(m.TotalPages),
	}
}

func BuildListResponse[D, R any](items []D, convert func(D) R, total int64, page, perPage int) ([]R, *openapi.PaginationMeta) {
	return lo.Map(items, func(item D, _ int) R { return convert(item) }), PaginationMeta(page, perPage, total)
}
