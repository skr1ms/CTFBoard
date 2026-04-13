package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type pageConstraints struct {
	Title   string `validate:"required,max=200"`
	Slug    string `validate:"required,max=100,page_slug"`
	Content string `validate:"max=50000"`
}

func ValidateCreatePageRequest(req *openapi.CreatePageRequest, v validator.Validator) error {
	c := pageConstraints{Title: req.Title, Slug: req.Slug, Content: lo.FromPtrOr(req.Content, "")}

	return ValidateConstraints(v, &c)
}

func ValidateUpdatePageRequest(req *openapi.UpdatePageRequest, v validator.Validator) error {
	c := pageConstraints{Title: req.Title, Slug: req.Slug, Content: lo.FromPtrOr(req.Content, "")}

	return ValidateConstraints(v, &c)
}

func CreatePageRequestToParams(req *openapi.CreatePageRequest) (title, slug, content string, isDraft bool, orderIndex int, err error) {
	return req.Title, req.Slug, lo.FromPtrOr(req.Content, ""), lo.FromPtrOr(req.IsDraft, true), lo.FromPtrOr(req.OrderIndex, 0), nil
}

func UpdatePageRequestToParams(req *openapi.UpdatePageRequest) (title, slug, content string, isDraft bool, orderIndex int, err error) {
	return req.Title, req.Slug, lo.FromPtrOr(req.Content, ""), lo.FromPtrOr(req.IsDraft, false), lo.FromPtrOr(req.OrderIndex, 0), nil
}
