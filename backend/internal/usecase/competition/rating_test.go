package competition

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRatingUseCase_GetGlobalRatings_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	page, perPage := 1, 20
	list := []*entity.GlobalRating{h.NewGlobalRating(uuid.New(), "Team1", 100, 2)}
	total := int64(1)

	deps.ratingRepo.EXPECT().GetGlobalRatings(mock.Anything, perPage, 0).Return(list, nil)
	deps.ratingRepo.EXPECT().CountGlobalRatings(mock.Anything).Return(total, nil)

	uc := h.CreateRatingUseCase()
	got, gotTotal, err := uc.GetGlobalRatings(ctx, page, perPage)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, total, gotTotal)
}

func TestRatingUseCase_GetGlobalRatings_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	page, perPage := 1, 20

	deps.ratingRepo.EXPECT().GetGlobalRatings(mock.Anything, perPage, 0).Return(nil, assert.AnError)

	uc := h.CreateRatingUseCase()
	got, gotTotal, err := uc.GetGlobalRatings(ctx, page, perPage)

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), gotTotal)
}

func TestRatingUseCase_GetTeamRating_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	teamID := uuid.New()
	global := h.NewGlobalRating(teamID, "Team1", 50, 1)
	teamRatings := []*entity.TeamRating{}

	deps.ratingRepo.EXPECT().GetGlobalRatingByTeamID(mock.Anything, teamID).Return(global, nil)
	deps.ratingRepo.EXPECT().GetTeamRatingsByTeamID(mock.Anything, teamID).Return(teamRatings, nil)

	uc := h.CreateRatingUseCase()
	gotGlobal, gotTeam, err := uc.GetTeamRating(ctx, teamID)

	assert.NoError(t, err)
	assert.NotNil(t, gotGlobal)
	assert.Equal(t, teamID, gotGlobal.TeamID)
	assert.Len(t, gotTeam, 0)
}

func TestRatingUseCase_GetTeamRating_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	teamID := uuid.New()

	deps.ratingRepo.EXPECT().GetGlobalRatingByTeamID(mock.Anything, teamID).Return(nil, assert.AnError)

	uc := h.CreateRatingUseCase()
	gotGlobal, gotTeam, err := uc.GetTeamRating(ctx, teamID)

	assert.Error(t, err)
	assert.Nil(t, gotGlobal)
	assert.Nil(t, gotTeam)
}

func TestRatingUseCase_GetCTFEvents_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	start := time.Now()
	end := start.Add(time.Hour)
	list := []*entity.CTFEvent{h.NewCTFEvent("event1", start, end, 1.0)}

	deps.ratingRepo.EXPECT().GetAllCTFEvents(mock.Anything).Return(list, nil)

	uc := h.CreateRatingUseCase()
	got, err := uc.GetCTFEvents(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "event1", got[0].Name)
}

func TestRatingUseCase_GetCTFEvents_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.ratingRepo.EXPECT().GetAllCTFEvents(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateRatingUseCase()
	got, err := uc.GetCTFEvents(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestRatingUseCase_CreateCTFEvent_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name := "event1"
	start := time.Now()
	end := start.Add(time.Hour)
	weight := 1.5

	deps.ratingRepo.EXPECT().CreateCTFEvent(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, e *entity.CTFEvent) {
		assert.Equal(t, name, e.Name)
		assert.Equal(t, start, e.StartTime)
		assert.Equal(t, end, e.EndTime)
		assert.Equal(t, weight, e.Weight)
	})

	uc := h.CreateRatingUseCase()
	got, err := uc.CreateCTFEvent(ctx, name, start, end, weight)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
}

func TestRatingUseCase_CreateCTFEvent_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name := "event1"
	start := time.Now()
	end := start.Add(time.Hour)

	deps.ratingRepo.EXPECT().CreateCTFEvent(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateRatingUseCase()
	got, err := uc.CreateCTFEvent(ctx, name, start, end, 1.0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestRatingUseCase_FinalizeCTFEvent_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	eventID := uuid.New()
	teamID := uuid.New()
	start := time.Now()
	end := start.Add(time.Hour)
	event := h.NewCTFEvent("e", start, end, 1.0)
	event.ID = eventID
	scoreboard := []*repo.ScoreboardEntry{h.NewScoreboardEntry(teamID, "T1", 100)}
	teams := []*entity.Team{{ID: teamID, Name: "T1"}}
	teamRatings := []*entity.TeamRating{{TeamID: teamID, CTFEventID: eventID, Rank: 1, Score: 100, RatingPoints: 10.0}}

	deps.ratingRepo.EXPECT().GetCTFEventByID(mock.Anything, eventID).Return(event, nil)
	deps.solveRepo.EXPECT().GetScoreboard(mock.Anything).Return(scoreboard, nil)
	deps.ratingRepo.EXPECT().CreateTeamRating(mock.Anything, mock.Anything).Return(nil)
	deps.teamRepo.EXPECT().GetAll(mock.Anything).Return(teams, nil)
	deps.ratingRepo.EXPECT().GetTeamRatingsByTeamID(mock.Anything, teamID).Return(teamRatings, nil)
	deps.ratingRepo.EXPECT().UpsertGlobalRating(mock.Anything, mock.Anything).Return(nil)

	uc := h.CreateRatingUseCase()
	err := uc.FinalizeCTFEvent(ctx, eventID)

	assert.NoError(t, err)
}

func TestRatingUseCase_FinalizeCTFEvent_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	eventID := uuid.New()

	deps.ratingRepo.EXPECT().GetCTFEventByID(mock.Anything, eventID).Return(nil, assert.AnError)

	uc := h.CreateRatingUseCase()
	err := uc.FinalizeCTFEvent(ctx, eventID)

	assert.Error(t, err)
}

func TestCalculateRatingPoints(t *testing.T) {
	points := CalculateRatingPoints(1, 10, 1.0)
	assert.Greater(t, points, 0.0)
	assert.Equal(t, 0.0, CalculateRatingPoints(0, 10, 1.0))
	assert.Equal(t, 0.0, CalculateRatingPoints(1, 0, 1.0))
}
