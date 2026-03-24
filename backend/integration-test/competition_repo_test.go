package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompetitionRepo_Get_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	repo := f.CompetitionRepo
	ctx := context.Background()

	comp, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, comp.ID)
	assert.NotEmpty(t, comp.Name, "competition has a name")
	if comp.StartTime != nil && comp.EndTime != nil {
		assert.True(t, comp.StartTime.Before(time.Now()), "competition should be started")
		assert.True(t, comp.EndTime.After(time.Now()), "competition should not be ended")
	}
}

func TestCompetitionRepo_Get_Error_CancelledContext(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.CompetitionRepo.Get(ctx)
	require.Error(t, err)
}

func TestCompetitionRepo_Update_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	repo := f.CompetitionRepo
	ctx := context.Background()

	comp, err := repo.Get(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := testPool.Pool.Exec(context.Background(),
			"UPDATE competition SET is_paused = false, is_public = true, name = 'CTF Competition', start_time = NOW() - INTERVAL '1 hour', updated_at = NOW() WHERE id = 1")
		if err != nil {
			panic("competition_repo cleanup: " + err.Error())
		}
	})

	now := time.Now().UTC().Truncate(time.Second)
	name := "Updated Name"
	comp.Name = name
	comp.StartTime = &now
	comp.IsPaused = true
	comp.IsPublic = false

	err = repo.Update(ctx, comp)
	require.NoError(t, err)

	updatedComp, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, name, updatedComp.Name)
	assert.NotNil(t, updatedComp.StartTime)
	assert.WithinDuration(t, now, *updatedComp.StartTime, time.Second)
	assert.True(t, updatedComp.IsPaused)
	assert.False(t, updatedComp.IsPublic)
}

func TestCompetitionRepo_Update_Partial(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	comp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)

	name := "Partial Update"
	freeze := time.Now().UTC().Add(1 * time.Hour).Truncate(time.Second)
	comp.Name = name
	comp.FreezeTime = &freeze

	err = f.CompetitionRepo.Update(ctx, comp)
	require.NoError(t, err)

	updatedComp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, name, updatedComp.Name)
	assert.Equal(t, freeze.Unix(), updatedComp.FreezeTime.Unix())
	assert.NotNil(t, updatedComp.StartTime, "seed sets start_time, partial update does not clear it")
}

func TestCompetitionRepo_Update_Error_CancelledContext(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	comp, err := f.CompetitionRepo.Get(context.Background())
	require.NoError(t, err)
	comp.Name = "Should Fail"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = f.CompetitionRepo.Update(ctx, comp)
	require.Error(t, err)
}
