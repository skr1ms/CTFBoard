package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRatingRepo_CreateCTFEvent_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	now := time.Now().UTC()
	event := &entity.CTFEvent{
		Name:      "CTF 2025",
		StartTime: now,
		EndTime:   now.Add(48 * time.Hour),
		Weight:    1.0,
	}
	err := f.RatingRepo.CreateCTFEvent(ctx, event)
	require.NoError(t, err)
	assert.NotEmpty(t, event.ID)
}

func TestRatingRepo_CreateCTFEvent_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	now := time.Now().UTC()
	event := &entity.CTFEvent{Name: "x", StartTime: now, EndTime: now, Weight: 1}
	err := f.RatingRepo.CreateCTFEvent(ctx, event)
	assert.Error(t, err)
}

func TestRatingRepo_GetCTFEventByID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	now := time.Now().UTC()
	event := &entity.CTFEvent{Name: "GetEvent", StartTime: now, EndTime: now.Add(24 * time.Hour), Weight: 1}
	err := f.RatingRepo.CreateCTFEvent(ctx, event)
	require.NoError(t, err)
	got, err := f.RatingRepo.GetCTFEventByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.ID, got.ID)
	assert.Equal(t, event.Name, got.Name)
}

func TestRatingRepo_GetCTFEventByID_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.RatingRepo.GetCTFEventByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrCTFEventNotFound))
}

func TestRatingRepo_GetAllCTFEvents_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	now := time.Now().UTC()
	event := &entity.CTFEvent{Name: "AllEvents", StartTime: now, EndTime: now, Weight: 1}
	err := f.RatingRepo.CreateCTFEvent(ctx, event)
	require.NoError(t, err)
	list, err := f.RatingRepo.GetAllCTFEvents(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestRatingRepo_GetAllCTFEvents_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.RatingRepo.GetAllCTFEvents(ctx)
	assert.Error(t, err)
}

func TestRatingRepo_CreateTeamRating_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	now := time.Now().UTC()
	event := &entity.CTFEvent{Name: "TeamRatingEvent", StartTime: now, EndTime: now, Weight: 1}
	err := f.RatingRepo.CreateCTFEvent(ctx, event)
	require.NoError(t, err)
	_, team := f.CreateUserWithTeam(t, "tr")
	tr := &entity.TeamRating{
		TeamID:       team.ID,
		CTFEventID:   event.ID,
		Rank:         1,
		Score:        100,
		RatingPoints: 50,
	}
	err = f.RatingRepo.CreateTeamRating(ctx, tr)
	require.NoError(t, err)
	assert.NotEmpty(t, tr.ID)
}

func TestRatingRepo_CreateTeamRating_Error_InvalidTeamID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	now := time.Now().UTC()
	event := &entity.CTFEvent{Name: "ErrTeamRating", StartTime: now, EndTime: now, Weight: 1}
	err := f.RatingRepo.CreateCTFEvent(ctx, event)
	require.NoError(t, err)
	tr := &entity.TeamRating{
		TeamID:       uuid.New(),
		CTFEventID:   event.ID,
		Rank:         1,
		Score:        100,
		RatingPoints: 50,
	}
	err = f.RatingRepo.CreateTeamRating(ctx, tr)
	assert.Error(t, err)
}
