package integration_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func applyCompetitionMigration(t *testing.T, db *sql.DB) {
	path := filepath.Join("..", "migrations", "000002_competition.up.sql")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	stmts := strings.Split(string(content), ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err = db.Exec(stmt)
		require.NoError(t, err)
	}
}

func TestCompetitionRepo_Get(t *testing.T) {
	testDB := SetupTestDB(t)
	applyCompetitionMigration(t, testDB.DB)
	repo := persistent.NewCompetitionRepo(testDB.DB)
	ctx := context.Background()

	comp, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, comp.Id)
	assert.Equal(t, "CTF Competition", comp.Name)
	assert.Nil(t, comp.StartTime)
	assert.Nil(t, comp.EndTime)
	assert.Nil(t, comp.FreezeTime)
}

func TestCompetitionRepo_Update(t *testing.T) {
	testDB := SetupTestDB(t)
	applyCompetitionMigration(t, testDB.DB)
	repo := persistent.NewCompetitionRepo(testDB.DB)
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
	applyCompetitionMigration(t, testDB.DB)
	repo := persistent.NewCompetitionRepo(testDB.DB)
	ctx := context.Background()

	comp, err := repo.Get(ctx)
	require.NoError(t, err)

	// Update only name and freeze time
	name := "Partial Update"
	freeze := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	comp.Name = name
	comp.FreezeTime = &freeze

	err = repo.Update(ctx, comp)
	require.NoError(t, err)

	updatedComp, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, name, updatedComp.Name)
	assert.Equal(t, freeze.Unix(), updatedComp.FreezeTime.Unix())
	// Other fields should remain as they were in the struct passed to Update.
	// Note: The Update method updates ALL fields based on the struct.
	// So fields not set in `comp` (if they were nil/zero) will be updated to nil/zero.
	// In this test, we reused `comp` from `Get`, so other fields like StartTime (nil) remain nil.
	assert.Nil(t, updatedComp.StartTime)
}
