package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func CreatePageRequestToParams(req *openapi.CreatePageRequest) (usecase.PageCreateParams, error) {
	return usecase.PageCreateParams{
		Title:      req.Title,
		Slug:       req.Slug,
		Content:    lo.FromPtrOr(req.Content, ""),
		IsDraft:    lo.FromPtrOr(req.IsDraft, true),
		OrderIndex: lo.FromPtrOr(req.OrderIndex, 0),
	}, nil
}

func UpdatePageRequestToParams(req *openapi.UpdatePageRequest) (usecase.PageUpdateParams, error) {
	return usecase.PageUpdateParams{
		Title:      req.Title,
		Slug:       req.Slug,
		Content:    lo.FromPtrOr(req.Content, ""),
		IsDraft:    lo.FromPtrOr(req.IsDraft, false),
		OrderIndex: lo.FromPtrOr(req.OrderIndex, 0),
	}, nil
}
