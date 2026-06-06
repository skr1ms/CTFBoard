package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateTagRequestToParams(req *openapi.CreateTagRequest) (name, color string, err error) {
	return req.Name, lo.FromPtrOr(req.Color, ""), nil
}

func UpdateTagRequestToParams(req *openapi.UpdateTagRequest) (name, color string, err error) {
	return req.Name, lo.FromPtrOr(req.Color, ""), nil
}
