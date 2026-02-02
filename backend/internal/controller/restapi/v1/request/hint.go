package request

import "github.com/skr1ms/CTFBoard/internal/openapi"

func CreateHintRequestToParams(req *openapi.RequestCreateHintRequest) (content string, cost, orderIndex int) {
	cost, orderIndex = 0, 0
	if req.Cost != nil {
		cost = *req.Cost
	}
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}
	return req.Content, cost, orderIndex
}

func UpdateHintRequestToParams(req *openapi.RequestUpdateHintRequest) (content string, cost, orderIndex int) {
	cost, orderIndex = 0, 0
	if req.Cost != nil {
		cost = *req.Cost
	}
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}
	return req.Content, cost, orderIndex
}
