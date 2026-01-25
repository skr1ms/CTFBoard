package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

type AwardResponse struct {
	Id          string  `json:"id"`
	TeamId      string  `json:"team_id"`
	Value       int     `json:"value"`
	Description string  `json:"description"`
	CreatedBy   *string `json:"created_by,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func FromAward(a *entity.Award) AwardResponse {
	res := AwardResponse{
		Id:          a.Id.String(),
		TeamId:      a.TeamId.String(),
		Value:       a.Value,
		Description: a.Description,
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
	}
	if a.CreatedBy != nil {
		cb := a.CreatedBy.String()
		res.CreatedBy = &cb
	}
	return res
}

func FromAwardList(items []*entity.Award) []AwardResponse {
	res := make([]AwardResponse, len(items))
	for i, item := range items {
		res[i] = FromAward(item)
	}
	return res
}
