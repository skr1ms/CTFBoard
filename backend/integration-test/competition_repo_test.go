package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Get Tests

func TestCompetitionRepo_Get(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	repo := f.CompetitionRepo
	ctx := context.Background()

	comp, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, comp.Id)
	assert.Equal(t, "CTF Competition", comp.Name)
	assert.Nil(t, comp.StartTime)
	assert.Nil(t, comp.EndTime)
	assert.Nil(t, comp.FreezeTime)
}

// Update Tests

func TestCompetitionRepo_Update(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	repo := f.CompetitionRepo
	ctx := context.Background()

	comp, err := repo.Get(ctx)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)
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
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	comp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)

	name := "Partial Update"
	freeze := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	comp.Name = name
	comp.FreezeTime = &freeze

	err = f.CompetitionRepo.Update(ctx, comp)
	require.NoError(t, err)

	updatedComp, err := f.CompetitionRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, name, updatedComp.Name)
	assert.Equal(t, freeze.Unix(), updatedComp.FreezeTime.Unix())
	assert.Nil(t, updatedComp.StartTime)
}
