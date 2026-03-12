package request

import (
	"regexp"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxPageTitleLength   = 200
	maxPageSlugLength    = 100
	maxPageContentLength = 50000
)

var pageSlugPattern = regexp.MustCompile(`^[a-z0-9\-]+$`)

func CreatePageRequestToParams(req *openapi.CreatePageRequest) (title, slug, content string, isDraft bool, orderIndex int, err error) {
	content = derefOr(req.Content, "")
	if len(req.Title) > maxPageTitleLength {
		return "", "", "", false, 0, helper.NewValidationErrorf("title must be at most %d characters", maxPageTitleLength)
	}
	if len(req.Slug) > maxPageSlugLength {
		return "", "", "", false, 0, helper.NewValidationErrorf("slug must be at most %d characters", maxPageSlugLength)
	}
	if !pageSlugPattern.MatchString(req.Slug) {
		return "", "", "", false, 0, helper.NewValidationErrorf("slug must match pattern %q", pageSlugPattern.String())
	}
	if len(content) > maxPageContentLength {
		return "", "", "", false, 0, helper.NewValidationErrorf("content must be at most %d characters", maxPageContentLength)
	}
	return req.Title, req.Slug, content, derefOr(req.IsDraft, true), derefOr(req.OrderIndex, 0), nil
}

func UpdatePageRequestToParams(req *openapi.UpdatePageRequest) (title, slug, content string, isDraft bool, orderIndex int, err error) {
	content = derefOr(req.Content, "")
	if len(req.Title) > maxPageTitleLength {
		return "", "", "", false, 0, helper.NewValidationErrorf("title must be at most %d characters", maxPageTitleLength)
	}
	if len(req.Slug) > maxPageSlugLength {
		return "", "", "", false, 0, helper.NewValidationErrorf("slug must be at most %d characters", maxPageSlugLength)
	}
	if !pageSlugPattern.MatchString(req.Slug) {
		return "", "", "", false, 0, helper.NewValidationErrorf("slug must match pattern %q", pageSlugPattern.String())
	}
	if len(content) > maxPageContentLength {
		return "", "", "", false, 0, helper.NewValidationErrorf("content must be at most %d characters", maxPageContentLength)
	}
	return req.Title, req.Slug, content, derefOr(req.IsDraft, false), derefOr(req.OrderIndex, 0), nil
}
