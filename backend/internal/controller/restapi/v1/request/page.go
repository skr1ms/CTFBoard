package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreatePageRequestToParams(req *openapi.CreatePageRequest) (title, slug, content string, isDraft bool, orderIndex int, err error) {
	return req.Title, req.Slug, lo.FromPtrOr(req.Content, ""), lo.FromPtrOr(req.IsDraft, true), lo.FromPtrOr(req.OrderIndex, 0), nil
}

func UpdatePageRequestToParams(req *openapi.UpdatePageRequest) (title, slug, content string, isDraft bool, orderIndex int, err error) {
	return req.Title, req.Slug, lo.FromPtrOr(req.Content, ""), lo.FromPtrOr(req.IsDraft, false), lo.FromPtrOr(req.OrderIndex, 0), nil
}
