package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateHintRequestToParams(req *openapi.CreateHintRequest) (content string, cost, orderIndex int, err error) {
	cost = derefOr(req.Cost, 0)
	orderIndex = derefOr(req.OrderIndex, 0)
	if cost < 0 {
		return "", 0, 0, helper.NewValidationErrorf("cost must be >= 0")
	}
	if orderIndex < 0 {
		return "", 0, 0, helper.NewValidationErrorf("order_index must be >= 0")
	}
	return req.Content, cost, orderIndex, nil
}

func UpdateHintRequestToParams(req *openapi.UpdateHintRequest) (content string, cost, orderIndex int, err error) {
	cost = derefOr(req.Cost, 0)
	orderIndex = derefOr(req.OrderIndex, 0)
	if cost < 0 {
		return "", 0, 0, helper.NewValidationErrorf("cost must be >= 0")
	}
	if orderIndex < 0 {
		return "", 0, 0, helper.NewValidationErrorf("order_index must be >= 0")
	}
	return req.Content, cost, orderIndex, nil
}
