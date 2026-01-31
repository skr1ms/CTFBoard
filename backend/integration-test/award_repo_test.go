package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAwardRepo_Create(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_c")
	_, team := f.CreateUserWithTeam(t, "team_create")

	award := &entity.Award{
		TeamID:      team.ID,
		Value:       100,
		Description: "Test Bonus",
		CreatedBy:   &admin.ID,
	}

	err := f.AwardRepo.Create(ctx, award)
	require.NoError(t, err)
	assert.NotZero(t, award.ID)
	assert.NotZero(t, award.CreatedAt)
}

func TestAwardRepo_GetByTeamID(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_g")
	_, team := f.CreateUserWithTeam(t, "team_get")

	award1 := &entity.Award{TeamID: team.ID, Value: 10, Description: "First", CreatedBy: &admin.ID}
	err := f.AwardRepo.Create(ctx, award1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	award2 := &entity.Award{TeamID: team.ID, Value: 20, Description: "Second", CreatedBy: &admin.ID}
	err = f.AwardRepo.Create(ctx, award2)
	require.NoError(t, err)

	awards, err := f.AwardRepo.GetByTeamID(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, awards, 2)

	assert.Equal(t, award2.ID, awards[0].ID)
	assert.Equal(t, award1.ID, awards[1].ID)
	assert.Equal(t, "Second", awards[0].Description)
	assert.NotNil(t, awards[0].CreatedBy)
	assert.Equal(t, admin.ID, *awards[0].CreatedBy)
}

func TestAwardRepo_GetTeamTotalAwards(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_t")
	_, team := f.CreateUserWithTeam(t, "team_total")

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, total)

	award1 := &entity.Award{TeamID: team.ID, Value: 100, Description: "Win", CreatedBy: &admin.ID}
	err = f.AwardRepo.Create(ctx, award1)
	require.NoError(t, err)

	award2 := &entity.Award{TeamID: team.ID, Value: -30, Description: "Penalty", CreatedBy: &admin.ID}
	err = f.AwardRepo.Create(ctx, award2)
	require.NoError(t, err)

	total, err = f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 70, total)
}
