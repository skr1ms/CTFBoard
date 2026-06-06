package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateCommentRequestToParams(req *openapi.CreateCommentRequest) (string, error) {
	return req.Content, nil
}
