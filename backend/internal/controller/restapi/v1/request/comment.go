package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const maxCommentContentLength = 2000

func CreateCommentRequestToParams(req *openapi.CreateCommentRequest) (string, error) {
	if len(req.Content) > maxCommentContentLength {
		return "", helper.NewValidationErrorf("content must be at most %d characters", maxCommentContentLength)
	}
	return req.Content, nil
}
