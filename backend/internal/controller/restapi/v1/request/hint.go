package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateHintRequestToParams(req *openapi.CreateHintRequest) (title, content string, cost, orderIndex int, err error) {
	return hintFieldsToParams(req.Cost, req.OrderIndex, req.Title, req.Content)
}

func UpdateHintRequestToParams(req *openapi.UpdateHintRequest) (title, content string, cost, orderIndex int, err error) {
	return hintFieldsToParams(req.Cost, req.OrderIndex, req.Title, req.Content)
}

func hintFieldsToParams(cost, orderIndex *int, title *string, content string) (string, string, int, int, error) {
	c := lo.FromPtrOr(cost, 0)
	oi := lo.FromPtrOr(orderIndex, 0)

	return lo.FromPtrOr(title, ""), content, c, oi, nil
}
