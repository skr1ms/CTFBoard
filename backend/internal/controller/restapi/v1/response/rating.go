package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromGlobalRating(g *entity.GlobalRating) openapi.ResponseGlobalRatingResponse {
	res := openapi.ResponseGlobalRatingResponse{
		TeamID:      ptr(g.TeamID.String()),
		TeamName:    ptr(g.TeamName),
		TotalPoints: ptr(float32(g.TotalPoints)),
		EventsCount: ptr(g.EventsCount),
		LastUpdated: ptr(g.LastUpdated),
	}
	if g.BestRank != nil {
		res.BestRank = ptr(*g.BestRank)
	}
	return res
}

func FromGlobalRatingsList(items []*entity.GlobalRating, total int64) openapi.ResponseGlobalRatingsListResponse {
	resItems := make([]openapi.ResponseGlobalRatingResponse, len(items))
	for i, item := range items {
		resItems[i] = FromGlobalRating(item)
	}
	return openapi.ResponseGlobalRatingsListResponse{
		Items: &resItems,
		Total: ptr(int(total)),
	}
}

func FromTeamRatingItem(tr *entity.TeamRating) openapi.ResponseTeamRatingItemResponse {
	return openapi.ResponseTeamRatingItemResponse{
		ID:           ptr(tr.ID.String()),
		CtfEventID:   ptr(tr.CTFEventID.String()),
		Rank:         ptr(tr.Rank),
		Score:        ptr(tr.Score),
		RatingPoints: ptr(float32(tr.RatingPoints)),
		CreatedAt:    ptr(tr.CreatedAt),
	}
}

func FromTeamRating(global *entity.GlobalRating, eventRatings []*entity.TeamRating) openapi.ResponseTeamRatingResponse {
	res := openapi.ResponseTeamRatingResponse{}
	if global != nil {
		g := FromGlobalRating(global)
		res.Global = &g
	}
	if len(eventRatings) > 0 {
		items := make([]openapi.ResponseTeamRatingItemResponse, len(eventRatings))
		for i, tr := range eventRatings {
			items[i] = FromTeamRatingItem(tr)
		}
		res.EventRatings = &items
	}
	return res
}

func FromCTFEvent(e *entity.CTFEvent) openapi.ResponseCTFEventResponse {
	return openapi.ResponseCTFEventResponse{
		ID:        ptr(e.ID.String()),
		Name:      ptr(e.Name),
		StartTime: ptr(e.StartTime),
		EndTime:   ptr(e.EndTime),
		Weight:    ptr(float32(e.Weight)),
		CreatedAt: ptr(e.CreatedAt),
	}
}

func FromCTFEventList(items []*entity.CTFEvent) []openapi.ResponseCTFEventResponse {
	res := make([]openapi.ResponseCTFEventResponse, len(items))
	for i, e := range items {
		res[i] = FromCTFEvent(e)
	}
	return res
}
