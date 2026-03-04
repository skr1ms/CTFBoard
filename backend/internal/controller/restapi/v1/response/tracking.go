package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromTrackingEntry(e *entity.TrackingEntry) openapi.TrackingEntry {
	t := e.TrackedAt
	return openapi.TrackingEntry{
		ID:        ptr(e.ID.String()),
		UserID:    ptr(e.UserID.String()),
		IP:        ptr(e.IP),
		UserAgent: ptr(e.UserAgent),
		TrackedAt: &t,
	}
}

func FromTrackingList(items []*entity.TrackingEntry, total int64, page, perPage int) openapi.TrackingListResponse {
	res := make([]openapi.TrackingEntry, len(items))
	for i, item := range items {
		res[i] = FromTrackingEntry(item)
	}
	return openapi.TrackingListResponse{
		Data: &res,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}
