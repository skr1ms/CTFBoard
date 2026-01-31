package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromAward(a *entity.Award) openapi.ResponseAwardResponse {
	res := openapi.ResponseAwardResponse{
		ID:          ptr(a.ID.String()),
		TeamID:      ptr(a.TeamID.String()),
		Value:       ptr(a.Value),
		Description: ptr(a.Description),
		CreatedAt:   ptr(a.CreatedAt.Format(time.RFC3339)),
	}
	if a.CreatedBy != nil {
		cb := a.CreatedBy.String()
		res.CreatedBy = &cb
	}
	return res
}

func FromAwardList(items []*entity.Award) []openapi.ResponseAwardResponse {
	res := make([]openapi.ResponseAwardResponse, len(items))
	for i, item := range items {
		res[i] = FromAward(item)
	}
	return res
}
