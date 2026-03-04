package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreatePageRequestToParams(req *openapi.CreatePageRequest) (title, slug, content string, isDraft bool, orderIndex int) {
	return req.Title, req.Slug,
		derefOr(req.Content, ""),
		derefOr(req.IsDraft, true),
		derefOr(req.OrderIndex, 0)
}

func UpdatePageRequestToParams(req *openapi.UpdatePageRequest) (title, slug, content string, isDraft bool, orderIndex int) {
	return req.Title, req.Slug,
		derefOr(req.Content, ""),
		derefOr(req.IsDraft, false),
		derefOr(req.OrderIndex, 0)
}
