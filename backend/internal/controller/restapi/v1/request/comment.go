package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type createCommentConstraints struct {
	Content string `validate:"required,max=2000"`
}

func ValidateCreateCommentRequest(req *openapi.CreateCommentRequest, v validator.Validator) error {
	c := createCommentConstraints{Content: req.Content}

	return ValidateConstraints(v, &c)
}

func CreateCommentRequestToParams(req *openapi.CreateCommentRequest) (string, error) {
	return req.Content, nil
}
