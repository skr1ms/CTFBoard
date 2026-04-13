package challenge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestAdminUpsertSolution_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	content := "## Solution\nThis is the writeup."

	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &repo.ChallengeSolution{ChallengeID: challengeID, Content: content, Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, content).Return(solution, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, challengeID, result.ChallengeID)
	assert.Equal(t, content, result.Content)
}

func TestAdminUpsertSolution_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, "content")

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
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "content").Return(nil, errors.New("db error"))

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, "content")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAdminUpsertSolution_EmptyContent(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &repo.ChallengeSolution{ChallengeID: challengeID, Content: "", Files: []*domain.File{}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "").Return(solution, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestAdminUpsertSolution_ContentTooLong(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	content := strings.Repeat("x", 524289)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, content)

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

	entries := []*repo.ChallengeSolutionEntry{
		{ChallengeID: cid1, ChallengeTitle: "Web 1", ChallengeCategory: "web", Content: "## WS1", Files: []*domain.File{}},
		{ChallengeID: cid2, ChallengeTitle: "Pwn 1", ChallengeCategory: "pwn", Content: "## PS1", Files: []*domain.File{}},
	}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("ListSolutions", mock.Anything, teamID).Return(entries, nil)

	result, err := uc.ListSolutions(context.Background(), teamID)

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
	d.challengeRepo.On("ListSolutions", mock.Anything, teamID).Return([]*repo.ChallengeSolutionEntry{}, nil)

	result, err := uc.ListSolutions(context.Background(), teamID)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestListSolutions_RepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("ListSolutions", mock.Anything, teamID).Return(nil, errors.New("db error"))

	result, err := uc.ListSolutions(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetSolution_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solve := &domain.Solve{ID: uuid.New(), TeamID: teamID, ChallengeID: challengeID}
	solution := &repo.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", Files: []*domain.File{}}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(solve, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

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

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

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

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrNotAuthenticated)
}

func TestGetSolution_NotSolved_Forbidden(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := newTestChallenge(challengeID, "Web Chall", "web", 100, "hash")

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

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
	solve := &domain.Solve{ID: uuid.New(), TeamID: teamID, ChallengeID: challengeID}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(solve, nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
