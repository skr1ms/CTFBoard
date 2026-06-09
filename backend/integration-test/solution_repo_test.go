package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSolutionRepo_Upsert_Create(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_upsert_create_"+uuid.New().String()[:6], 100)

	sol, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## First Writeup", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, sol.ChallengeID)
	assert.Equal(t, "## First Writeup", sol.Content)
	assert.Equal(t, domain.SolutionStateSolvedOnly, sol.State)
}

func TestSolutionRepo_Upsert_Update(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_upsert_upd_"+uuid.New().String()[:6], 100)

	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Old Content", domain.SolutionStateHidden)
	require.NoError(t, err)

	updated, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Updated Content", domain.SolutionStateAfterEvent)
	require.NoError(t, err)
	assert.Equal(t, "## Updated Content", updated.Content)
	assert.Equal(t, domain.SolutionStateAfterEvent, updated.State)
}

func TestSolutionRepo_GetSolution_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_get_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Test Writeup", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, sol.ChallengeID)
	assert.Equal(t, "## Test Writeup", sol.Content)
	assert.Equal(t, domain.SolutionStateSolvedOnly, sol.State)
}

func TestSolutionRepo_GetSolution_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_notfound_"+uuid.New().String()[:6], 100)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	assert.Error(t, err)
	assert.Nil(t, sol)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestSolutionRepo_GetSolution_WithWriteupFiles(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_with_files_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## With Files", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)

	writeupFile := &domain.File{
		Type:        domain.FileTypeWriteup,
		ChallengeID: &challenge.ID,
		Location:    "uploads/test-writeup.pdf",
		Filename:    "writeup.pdf",
		Size:        1024,
		SHA256:      "abc123",
	}
	err = f.FileRepo.Create(ctx, writeupFile)
	require.NoError(t, err)

	challengeFile := &domain.File{
		Type:        domain.FileTypeChallenge,
		ChallengeID: &challenge.ID,
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
	assert.Equal(t, domain.FileTypeWriteup, sol.Files[0].Type)
	assert.Equal(t, "writeup.pdf", sol.Files[0].Filename)
}

func TestSolutionRepo_DeleteSolution_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_del_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Will be deleted", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)

	err = f.ChallengeRepo.DeleteSolution(ctx, challenge.ID)
	require.NoError(t, err)

	sol, err := f.ChallengeRepo.GetSolution(ctx, challenge.ID)
	assert.Error(t, err)
	assert.Nil(t, sol)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestSolutionRepo_DeleteSolution_Idempotent(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_del_idem_"+uuid.New().String()[:6], 100)

	err := f.ChallengeRepo.DeleteSolution(ctx, challenge.ID)
	assert.NoError(t, err)
}

func TestSolutionRepo_DeleteChallengeDeletesSolution(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_cascade_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## Cascade test", domain.SolutionStateSolvedOnly)
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

func TestSolutionRepo_ListSolutions_ReturnsVisibleCandidates(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	cSolved := f.CreateChallenge(t, "sol_lst_solved_"+uuid.New().String()[:6], 100)
	cUnsolved := f.CreateChallenge(t, "sol_lst_unsolv_"+uuid.New().String()[:6], 200)

	_, err := f.ChallengeRepo.UpsertSolution(ctx, cSolved.ID, "## Solved writeup", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)
	_, err = f.ChallengeRepo.UpsertSolution(ctx, cUnsolved.ID, "## Unsolved writeup", domain.SolutionStateHidden)
	require.NoError(t, err)

	entries, err := f.ChallengeRepo.ListSolutions(ctx)
	require.NoError(t, err)

	solvedEntry := requireSolutionEntry(t, entries, cSolved.ID)
	assert.Equal(t, "## Solved writeup", solvedEntry.Content)
	assert.Equal(t, domain.SolutionStateSolvedOnly, solvedEntry.State)

	hiddenEntry := requireSolutionEntry(t, entries, cUnsolved.ID)
	assert.Equal(t, "## Unsolved writeup", hiddenEntry.Content)
	assert.Equal(t, domain.SolutionStateHidden, hiddenEntry.State)
}

func TestSolutionRepo_ListSolutions_ExcludesHiddenChallenges(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_lst_e_chall_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## hidden challenge content", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE challenges SET state = $1 WHERE id = $2", domain.ChallengeStateHidden, challenge.ID)
	require.NoError(t, err)

	entries, err := f.ChallengeRepo.ListSolutions(ctx)
	require.NoError(t, err)
	assert.Nil(t, findSolutionEntry(entries, challenge.ID))
}

func TestSolutionRepo_ListSolutions_IncludesWriteupFiles(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "sol_lst_f_chall_"+uuid.New().String()[:6], 100)
	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "## with files", domain.SolutionStateSolvedOnly)
	require.NoError(t, err)

	wFile := &domain.File{
		Type:        domain.FileTypeWriteup,
		ChallengeID: &challenge.ID,
		Location:    "uploads/sol.pdf",
		Filename:    "sol.pdf",
		Size:        512,
		SHA256:      "aaa",
	}
	require.NoError(t, f.FileRepo.Create(ctx, wFile))

	entries, err := f.ChallengeRepo.ListSolutions(ctx)
	require.NoError(t, err)
	entry := requireSolutionEntry(t, entries, challenge.ID)
	require.Len(t, entry.Files, 1)
	assert.Equal(t, "sol.pdf", entry.Files[0].Filename)
}

func TestSolutionRepo_UpsertSolution_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	challenge := f.CreateChallenge(t, "sol_ctx_"+uuid.New().String()[:6], 100)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.ChallengeRepo.UpsertSolution(ctx, challenge.ID, "content", domain.SolutionStateSolvedOnly)
	assert.Error(t, err)
}

func requireSolutionEntry(t *testing.T, entries []*domain.ChallengeSolutionEntry, challengeID uuid.UUID) *domain.ChallengeSolutionEntry {
	t.Helper()

	entry := findSolutionEntry(entries, challengeID)
	require.NotNil(t, entry, "missing solution entry for challenge %s", challengeID)

	return entry
}

func findSolutionEntry(entries []*domain.ChallengeSolutionEntry, challengeID uuid.UUID) *domain.ChallengeSolutionEntry {
	for _, entry := range entries {
		if entry.ChallengeID == challengeID {
			return entry
		}
	}

	return nil
}
