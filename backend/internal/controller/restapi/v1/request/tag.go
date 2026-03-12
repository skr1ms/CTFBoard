package request

import (
	"regexp"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validTagColor(color string) bool {
	return color == "" || hexColorRe.MatchString(color)
}

func CreateTagRequestToParams(req *openapi.CreateTagRequest) (name, color string, err error) {
	if req.Color != nil {
		color = *req.Color
	}
	if !validTagColor(color) {
		return "", "", helper.NewValidationErrorf("color must be empty or a hex color (#RRGGBB)")
	}
	return req.Name, color, nil
}

func UpdateTagRequestToParams(req *openapi.UpdateTagRequest) (name, color string, err error) {
	if req.Color != nil {
		color = *req.Color
	}
	if !validTagColor(color) {
		return "", "", helper.NewValidationErrorf("color must be empty or a hex color (#RRGGBB)")
	}
	return req.Name, color, nil
}
