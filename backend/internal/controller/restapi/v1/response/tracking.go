package response

import (
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromTrackingEntry(e *domain.TrackingEntry) openapi.TrackingEntry {
	t := e.TrackedAt
	return openapi.TrackingEntry{
		ID:        httputil.Ptr(e.ID.String()),
		UserID:    httputil.Ptr(e.UserID.String()),
		IP:        httputil.Ptr(e.IP),
		UserAgent: httputil.Ptr(e.UserAgent),
		TrackedAt: &t,
	}
}

func FromTrackingList(items []*domain.TrackingEntry, total int64, page, perPage int) openapi.TrackingListResponse {
	data, meta := BuildListResponse(items, FromTrackingEntry, total, page, perPage)
	return openapi.TrackingListResponse{Data: &data, Meta: meta}
}
