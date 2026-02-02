package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAwardRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_getall")
	_, team1 := f.CreateUserWithTeam(t, "team_getall_1")
	_, team2 := f.CreateUserWithTeam(t, "team_getall_2")

	a1 := &entity.Award{TeamID: team1.ID, Value: 10, Description: "A1", CreatedBy: &admin.ID}
	a2 := &entity.Award{TeamID: team2.ID, Value: 20, Description: "A2", CreatedBy: &admin.ID}
	require.NoError(t, f.AwardRepo.Create(ctx, a1))
	require.NoError(t, f.AwardRepo.Create(ctx, a2))

	awards, err := f.AwardRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(awards), 2)
}

func TestAwardRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	awards, err := f.AwardRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, awards)
}

func TestAwardRepo_Create_Success(t *testing.T) {
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

func TestAwardRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Helper()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	admin := f.CreateUser(t, "admin_ctx")
	_, team := f.CreateUserWithTeam(t, "team_ctx")

	award := &entity.Award{
		TeamID:      team.ID,
		Value:       10,
		Description: "Fail",
		CreatedBy:   &admin.ID,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.AwardRepo.Create(ctx, award)
	assert.Error(t, err)
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
