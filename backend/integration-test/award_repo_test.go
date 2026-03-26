package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestAwardRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_getall")
	_, team1 := f.CreateUserWithTeam(t, "team_getall_1")
	_, team2 := f.CreateUserWithTeam(t, "team_getall_2")

	award1 := f.CreateAward(t, team1.ID, 10, "A1", &admin.ID)
	award2 := f.CreateAward(t, team2.ID, 20, "A2", &admin.ID)

	awards, err := f.AwardRepo.GetAll(ctx)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, a := range awards {
		ids[a.ID] = true
	}

	assert.True(t, ids[award1.ID], "award1 should be in GetAll result")
	assert.True(t, ids[award2.ID], "award2 should be in GetAll result")
}

func TestAwardRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	awards, err := f.AwardRepo.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, awards)
}

func TestAwardRepo_Create_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	admin := f.CreateUser(t, "admin_c")
	_, team := f.CreateUserWithTeam(t, "team_create")

	award := f.CreateAward(t, team.ID, 100, "Test Bonus", &admin.ID)
	assert.NotZero(t, award.ID)
	assert.NotZero(t, award.CreatedAt)
}

func TestAwardRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	admin := f.CreateUser(t, "admin_ctx")
	_, team := f.CreateUserWithTeam(t, "team_ctx")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		award := &domain.Award{
			TeamID:      team.ID,
			Value:       10,
			Description: "Fail",
			CreatedBy:   &admin.ID,
		}

		return f.AwardRepo.Create(txCtx, award)
	})
	assert.Error(t, err)
}

func TestAwardRepo_GetByTeamID_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_g")
	_, team := f.CreateUserWithTeam(t, "team_get")

	award1 := f.CreateAward(t, team.ID, 10, "First", &admin.ID)
	award2 := f.CreateAward(t, team.ID, 20, "Second", &admin.ID)

	require.Eventually(t, func() bool {
		awards, err := f.AwardRepo.GetByTeamID(ctx, team.ID)

		return err == nil && len(awards) == 2 && awards[0].ID == award2.ID && awards[1].ID == award1.ID
	}, 2*time.Second, 10*time.Millisecond)

	awards, err := f.AwardRepo.GetByTeamID(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, awards, 2)

	assert.Equal(t, award2.ID, awards[0].ID)
	assert.Equal(t, award1.ID, awards[1].ID)
	assert.Equal(t, "Second", awards[0].Description)
	assert.NotNil(t, awards[0].CreatedBy)
	assert.Equal(t, admin.ID, *awards[0].CreatedBy)
}

func TestAwardRepo_GetByTeamID_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	_, team := f.CreateUserWithTeam(t, "team_get_err")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	awards, err := f.AwardRepo.GetByTeamID(ctx, team.ID)
	assert.Error(t, err)
	assert.Nil(t, awards)
}

func TestAwardRepo_GetTeamTotalAwards_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	admin := f.CreateUser(t, "admin_t")
	_, team := f.CreateUserWithTeam(t, "team_total")

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, total)

	f.CreateAward(t, team.ID, 100, "Win", &admin.ID)
	f.CreateAward(t, team.ID, -30, "Penalty", &admin.ID)

	total, err = f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, 70, total)
}

func TestAwardRepo_GetTeamTotalAwards_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	_, team := f.CreateUserWithTeam(t, "team_total_err")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.ID)
	assert.Error(t, err)
}
