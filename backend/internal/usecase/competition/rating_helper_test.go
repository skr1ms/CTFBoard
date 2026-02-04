package competition

import (
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *CompetitionTestHelper) CreateRatingUseCase() *RatingUseCase {
	h.t.Helper()
	return NewRatingUseCase(h.deps.ratingRepo, h.deps.solveRepo, h.deps.teamRepo)
}

func (h *CompetitionTestHelper) NewCTFEvent(name string, startTime, endTime time.Time, weight float64) *entity.CTFEvent {
	h.t.Helper()
	return &entity.CTFEvent{
		ID:        uuid.New(),
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
		Weight:    weight,
	}
}

func (h *CompetitionTestHelper) NewGlobalRating(teamID uuid.UUID, teamName string, totalPoints float64, eventsCount int) *entity.GlobalRating {
	h.t.Helper()
	return &entity.GlobalRating{
		TeamID:      teamID,
		TeamName:    teamName,
		TotalPoints: totalPoints,
		EventsCount: eventsCount,
		LastUpdated: time.Now(),
	}
}
