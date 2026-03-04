package request

import "github.com/TakuyaYagam1/AstroCTFb/internal/openapi"

func CreateHintRequestToParams(req *openapi.CreateHintRequest) (content string, cost, orderIndex int) {
	return req.Content, derefOr(req.Cost, 0), derefOr(req.OrderIndex, 0)
}

func UpdateHintRequestToParams(req *openapi.UpdateHintRequest) (content string, cost, orderIndex int) {
	return req.Content, derefOr(req.Cost, 0), derefOr(req.OrderIndex, 0)
}
