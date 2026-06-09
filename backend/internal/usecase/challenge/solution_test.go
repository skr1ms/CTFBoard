package challenge

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type solutionTestSettingsRepo struct {
	settings *domain.Settings
	err      error
}

func (r *solutionTestSettingsRepo) Get(context.Context) (*domain.Settings, error) {
	return r.settings, r.err
}

func (r *solutionTestSettingsRepo) GetForUpdate(context.Context) (*domain.Settings, error) {
	return r.settings, r.err
}

func (r *solutionTestSettingsRepo) Update(context.Context, *domain.Settings) error {
	return r.err
}

func (r *solutionTestSettingsRepo) UpdateIfCurrent(context.Context, *domain.Settings) error {
	return r.err
}

func solutionParams(content, state string) usecase.ChallengeSolutionUpsertParams {
	return usecase.ChallengeSolutionUpsertParams{Content: content, State: state}
}

func TestAdminUpsertSolution_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	content := "## Solution\nThis is the writeup."

	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: content, State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, content, domain.SolutionStateSolvedOnly).Return(solution, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, solutionParams(content, ""))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, challengeID, result.ChallengeID)
	assert.Equal(t, content, result.Content)
	assert.Equal(t, domain.SolutionStateSolvedOnly, result.State)
}

func TestAdminUpsertSolution_PreservesExistingStateWhenOmitted(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	existing := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "old", State: domain.SolutionStateHidden, Files: []*domain.File{}}
	updated := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "new", State: domain.SolutionStateHidden, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(existing, nil)
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "new", domain.SolutionStateHidden).Return(updated, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, solutionParams("new", ""))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "new", result.Content)
	assert.Equal(t, domain.SolutionStateHidden, result.State)
}

func TestAdminUpsertSolution_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, solutionParams("content", ""))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestAdminUpsertSolution_RepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "content", domain.SolutionStateSolvedOnly).Return(nil, errors.New("db error"))

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, solutionParams("content", ""))

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAdminUpsertSolution_EmptyContent(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "", State: domain.SolutionStateHidden, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "", domain.SolutionStateHidden).Return(solution, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, solutionParams("", domain.SolutionStateHidden))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Content)
	assert.Equal(t, domain.SolutionStateHidden, result.State)
}

func TestAdminUpsertSolution_ContentTooLong(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	content := strings.Repeat("x", 524289)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, solutionParams(content, ""))

	assert.Error(t, err)
	assert.Nil(t, result)

	var ve *apperr.ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAdminUpsertSolution_InvalidState(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	result, err := uc.AdminUpsertSolution(context.Background(), uuid.New(), solutionParams("content", "bad_state"))

	assert.Error(t, err)
	assert.Nil(t, result)

	var ve *apperr.ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAdminDeleteSolution_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("DeleteSolution", mock.Anything, challengeID).Return(nil)

	err := uc.AdminDeleteSolution(context.Background(), challengeID)

	assert.NoError(t, err)
}

func TestAdminDeleteSolution_RepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("DeleteSolution", mock.Anything, challengeID).Return(errors.New("db error"))

	err := uc.AdminDeleteSolution(context.Background(), challengeID)

	assert.Error(t, err)
}

func TestAdminDeleteSolution_NonExistent_IsIdempotent(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("DeleteSolution", mock.Anything, challengeID).Return(nil)

	err := uc.AdminDeleteSolution(context.Background(), challengeID)

	assert.NoError(t, err)
}

func TestListSolutions_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	cid1 := uuid.New()
	cid2 := uuid.New()

	entries := []*domain.ChallengeSolutionEntry{
		{ChallengeID: cid1, ChallengeTitle: "Web 1", ChallengeCategory: "web", Content: "## WS1", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}},
		{ChallengeID: cid2, ChallengeTitle: "Pwn 1", ChallengeCategory: "pwn", Content: "## PS1", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}},
	}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("ListSolutions", mock.Anything).Return(entries, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{cid1, cid2}, nil)

	result, err := uc.ListSolutions(context.Background(), &teamID, false)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Web 1", result[0].ChallengeTitle)
	assert.Equal(t, "Pwn 1", result[1].ChallengeTitle)
}

func TestListSolutions_Empty(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("ListSolutions", mock.Anything).Return([]*domain.ChallengeSolutionEntry{}, nil)

	result, err := uc.ListSolutions(context.Background(), &teamID, false)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestListSolutions_RepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("ListSolutions", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.ListSolutions(context.Background(), &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestListSolutions_FiltersInaccessibleStates(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	solvedID := uuid.New()
	unsolvedID := uuid.New()
	hiddenID := uuid.New()

	entries := []*domain.ChallengeSolutionEntry{
		{ChallengeID: solvedID, ChallengeTitle: "Solved", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}},
		{ChallengeID: unsolvedID, ChallengeTitle: "Unsolved", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}},
		{ChallengeID: hiddenID, ChallengeTitle: "Hidden", State: domain.SolutionStateHidden, Files: []*domain.File{}},
	}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("ListSolutions", mock.Anything).Return(entries, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{solvedID}, nil)

	result, err := uc.ListSolutions(context.Background(), &teamID, false)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, solvedID, result[0].ChallengeID)
}

func TestListSolutions_AfterEvent(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	entries := []*domain.ChallengeSolutionEntry{
		{ChallengeID: challengeID, ChallengeTitle: "After event", State: domain.SolutionStateAfterEvent, Files: []*domain.File{}},
	}

	d.challengeRepo.On("ListSolutions", mock.Anything).Return(entries, nil)
	d.compRepo.On("Get", mock.Anything).Return(&domain.Competition{StartTime: &start, EndTime: &end}, nil)

	result, err := uc.ListSolutions(context.Background(), nil, false)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, challengeID, result[0].ChallengeID)
}

func TestListSolutions_WriteupsDisabled(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()
	uc.deps.SettingsRepo = &solutionTestSettingsRepo{settings: &domain.Settings{WriteupEnabled: false}}

	d.challengeRepo.On("ListSolutions", mock.Anything).Return([]*domain.ChallengeSolutionEntry{
		{ChallengeID: uuid.New(), State: domain.SolutionStateAfterEvent},
	}, nil)

	result, err := uc.ListSolutions(context.Background(), nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrWriteupsDisabled)
}

func TestGetSolution_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solve := &domain.Solve{ID: uuid.New(), TeamID: teamID, ChallengeID: challengeID}
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(solve, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "## Solution", result.Content)
}

func TestGetSolution_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestGetSolution_NoTeamID_Forbidden(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrSolutionAccessDenied)
}

func TestGetSolution_NotSolved_Forbidden(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateSolvedOnly, Files: []*domain.File{}}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetSolution_SolutionNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetSolution_HiddenDeniedEvenWhenSolved(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateHidden, Files: []*domain.File{}}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrSolutionAccessDenied)
}

func TestGetSolution_AfterEventBeforeEndDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateAfterEvent, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)
	d.compRepo.On("Get", mock.Anything).Return(&domain.Competition{StartTime: &start, EndTime: &end}, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrSolutionAccessDenied)
}

func TestGetSolution_AfterEventAfterEndAllowed(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateAfterEvent, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)
	d.compRepo.On("Get", mock.Anything).Return(&domain.Competition{StartTime: &start, EndTime: &end}, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "## Solution", result.Content)
}

func TestGetSolution_WriteupsDisabledDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()
	uc.deps.SettingsRepo = &solutionTestSettingsRepo{settings: &domain.Settings{WriteupEnabled: false}}

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateAfterEvent, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrWriteupsDisabled)
}

func TestGetSolution_AdminBypassesStateAndWriteupSetting(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()
	uc.deps.SettingsRepo = &solutionTestSettingsRepo{settings: &domain.Settings{WriteupEnabled: false}}

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	challenge.State = domain.ChallengeStateHidden
	solution := &domain.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", State: domain.SolutionStateAdminOnly, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil, true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.SolutionStateAdminOnly, result.State)
}
