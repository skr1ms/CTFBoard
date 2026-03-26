package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func CreateHintRequestToParams(req *openapi.CreateHintRequest) (title, content string, cost, orderIndex int, err error) {
	cost = lo.FromPtrOr(req.Cost, 0)
	orderIndex = lo.FromPtrOr(req.OrderIndex, 0)

	if cost < 0 {
		return "", "", 0, 0, httperr.NewValidationErrorf("cost must be >= 0")
	}

	if orderIndex < 0 {
		return "", "", 0, 0, httperr.NewValidationErrorf("order_index must be >= 0")
	}

	return lo.FromPtrOr(req.Title, ""), req.Content, cost, orderIndex, nil
}

func UpdateHintRequestToParams(req *openapi.UpdateHintRequest) (title, content string, cost, orderIndex int, err error) {
	cost = lo.FromPtrOr(req.Cost, 0)
	orderIndex = lo.FromPtrOr(req.OrderIndex, 0)

	if cost < 0 {
		return "", "", 0, 0, httperr.NewValidationErrorf("cost must be >= 0")
	}

	if orderIndex < 0 {
		return "", "", 0, 0, httperr.NewValidationErrorf("order_index must be >= 0")
	}

	return lo.FromPtrOr(req.Title, ""), req.Content, cost, orderIndex, nil
}
