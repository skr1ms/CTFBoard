package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func TestSolutionRepo_Upsert_Create(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_upsert_create_"+uuid.New().String()[:6], 100)

	sol, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## First Writeup")
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, sol.ChallengeID)
	assert.Equal(t, "## First Writeup", sol.Content)
}

func TestSolutionRepo_Upsert_Update(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_upsert_upd_"+uuid.New().String()[:6], 100)

	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Old Content")
	require.NoError(t, err)

	updated, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Updated Content")
	require.NoError(t, err)
	assert.Equal(t, "## Updated Content", updated.Content)
}

func TestSolutionRepo_GetSolution_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_get_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Test Writeup")
	require.NoError(t, err)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, sol.ChallengeID)
	assert.Equal(t, "## Test Writeup", sol.Content)
}

func TestSolutionRepo_GetSolution_NotFound(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_notfound_"+uuid.New().String()[:6], 100)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	assert.Error(t, err)
	assert.Nil(t, sol)
	assert.ErrorIs(t, err, httperr.ErrChallengeNotFound)
}

func TestSolutionRepo_GetSolution_WithWriteupFiles(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_with_files_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## With Files")
	require.NoError(t, err)

	writeupFile := &entity.File{
		Type:        entity.FileTypeWriteup,
		ChallengeID: challenge.ID,
		Location:    "uploads/test-writeup.pdf",
		Filename:    "writeup.pdf",
		Size:        1024,
		SHA256:      "abc123",
	}
	err = f.FileRepo.Create(ctx, writeupFile)
	require.NoError(t, err)

	challengeFile := &entity.File{
		Type:        entity.FileTypeChallenge,
		ChallengeID: challenge.ID,
		Location:    "uploads/challenge.zip",
		Filename:    "challenge.zip",
		Size:        2048,
		SHA256:      "def456",
	}
	err = f.FileRepo.Create(ctx, challengeFile)
	require.NoError(t, err)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	require.NoError(t, err)
	require.Len(t, sol.Files, 1, "only writeup-type files should be included")
	assert.Equal(t, entity.FileTypeWriteup, sol.Files[0].Type)
	assert.Equal(t, "writeup.pdf", sol.Files[0].Filename)
}

func TestSolutionRepo_DeleteSolution_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_del_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Will be deleted")
	require.NoError(t, err)

	err = f.ChallengeRepo.DeleteSolution(ctx, challenge.ID)
	require.NoError(t, err)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	assert.Error(t, err)
	assert.Nil(t, sol)
	assert.ErrorIs(t, err, httperr.ErrChallengeNotFound)
}

func TestSolutionRepo_DeleteSolution_Idempotent(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_del_idem_"+uuid.New().String()[:6], 100)

	err := f.ChallengeRepo.DeleteSolution(ctx, challenge.ID)
	assert.NoError(t, err)
}

func TestSolutionRepo_DeleteChallengeDeletesSolution(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_cascade_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Cascade test")
	require.NoError(t, err)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	require.NoError(t, err)
	require.NotNil(t, sol)

	err = f.ChallengeRepo.Delete(ctx, challenge.ID)
	require.NoError(t, err)

	sol, err = f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	assert.Error(t, err)
	assert.Nil(t, sol)
}

func TestSolutionRepo_ListSolutions_ReturnsOnlySolvedByTeam(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "sol_list_"+uuid.New().String()[:6])

	cSolved := f.CreateChallenge(t, "sol_lst_solved_"+uuid.New().String()[:6], 100)
	cUnsolved := f.CreateChallenge(t, "sol_lst_unsolv_"+uuid.New().String()[:6], 200)

	_, err := f.ChallengeRepo.UpsertSolution(ctx, cSolved.ID, "## Solved writeup")
	require.NoError(t, err)
	_, err = f.ChallengeRepo.UpsertSolution(ctx, cUnsolved.ID, "## Unsolved writeup")
	require.NoError(t, err)

	f.CreateSolve(t, user.ID, team.ID, cSolved.ID)

	entries, err := f.ChallengeRepo.ListSolutions(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, cSolved.ID, entries[0].ChallengeID)
	assert.Equal(t, "## Solved writeup", entries[0].Content)
}

func TestSolutionRepo_ListSolutions_Empty(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "sol_lst_empty_"+uuid.New().String()[:6])
	challenge := f.CreateChallenge(t, "sol_lst_e_chall_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## some content")
	require.NoError(t, err)

	entries, err := f.ChallengeRepo.ListSolutions(ctx, team.ID)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSolutionRepo_ListSolutions_IncludesWriteupFiles(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "sol_lst_files_"+uuid.New().String()[:6])
	challenge := f.CreateChallenge(t, "sol_lst_f_chall_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## with files")
	require.NoError(t, err)
	f.CreateSolve(t, user.ID, team.ID, challenge.ID)

	wFile := &entity.File{
		Type:        entity.FileTypeWriteup,
		ChallengeID: challenge.ID,
		Location:    "uploads/sol.pdf",
		Filename:    "sol.pdf",
		Size:        512,
		SHA256:      "aaa",
	}
	require.NoError(t, f.FileRepo.Create(ctx, wFile))

	entries, err := f.ChallengeRepo.ListSolutions(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, entries[0].Files, 1)
	assert.Equal(t, "sol.pdf", entries[0].Files[0].Filename)
}

func TestSolutionRepo_UpsertSolution_CancelledContext(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "sol_ctx_"+uuid.New().String()[:6], 100)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "content")
	assert.Error(t, err)
}
