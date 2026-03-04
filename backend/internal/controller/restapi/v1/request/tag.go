package request

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func CreateTagRequestToParams(req *openapi.CreateTagRequest) (name, color string) {
	if req.Color != nil {
		color = *req.Color
	}
	return req.Name, color
}

func UpdateTagRequestToParams(req *openapi.UpdateTagRequest) (name, color string) {
	if req.Color != nil {
		color = *req.Color
	}
	return req.Name, color
}
