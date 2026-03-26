package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromTrackingEntry(e *domain.TrackingEntry) openapi.TrackingEntry {
	t := e.TrackedAt

	return openapi.TrackingEntry{
		ID:        new(e.ID.String()),
		UserID:    new(e.UserID.String()),
		IP:        new(e.IP),
		UserAgent: new(e.UserAgent),
		TrackedAt: &t,
	}
}

func FromTrackingList(items []*domain.TrackingEntry, total int64, page, perPage int) openapi.TrackingListResponse {
	data, meta := BuildListResponse(items, FromTrackingEntry, total, page, perPage)

	return openapi.TrackingListResponse{Data: &data, Meta: meta}
}
