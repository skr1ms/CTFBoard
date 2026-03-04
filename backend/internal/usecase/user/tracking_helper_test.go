package user

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mocks"
	"github.com/google/uuid"
)

type TrackingTestHelper struct {
	t            *testing.T
	TrackingRepo *mocks.MockTrackingRepository
}

func NewTrackingTestHelper(t *testing.T) *TrackingTestHelper {
	t.Helper()
	return &TrackingTestHelper{
		t:            t,
		TrackingRepo: mocks.NewMockTrackingRepository(t),
	}
}

func (h *TrackingTestHelper) CreateUseCase() *TrackingUseCase {
	h.t.Helper()
	return NewTrackingUseCase(TrackingDeps{TrackingRepo: h.TrackingRepo})
}

func (h *TrackingTestHelper) NewTrackingEntry(userID uuid.UUID, ip, userAgent string) *entity.TrackingEntry {
	h.t.Helper()
	return &entity.TrackingEntry{
		ID:        uuid.New(),
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
	}
}

func (h *TrackingTestHelper) NewChallengeOpen(userID, challengeID uuid.UUID, ip string) *entity.ChallengeOpen {
	h.t.Helper()
	return &entity.ChallengeOpen{
		UserID:      userID,
		ChallengeID: challengeID,
		IP:          ip,
	}
}
