package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAdminUpsertSolution_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	content := "## Solution\nThis is the writeup."

	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &repo.ChallengeSolution{ChallengeID: challengeID, Content: content, Files: []*entity.File{}}

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, content).Return(solution, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, challengeID, result.ChallengeID)
	assert.Equal(t, content, result.Content)
}

func TestAdminUpsertSolution_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, "content")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
}

func TestAdminUpsertSolution_RepoError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "content").Return(nil, errors.New("db error"))

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, "content")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAdminUpsertSolution_EmptyContent(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solution := &repo.ChallengeSolution{ChallengeID: challengeID, Content: "", Files: []*entity.File{}}

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("UpsertSolution", mock.Anything, challengeID, "").Return(solution, nil)

	result, err := uc.AdminUpsertSolution(context.Background(), challengeID, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Content)
}

func TestAdminDeleteSolution_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()

	deps.challengeRepo.On("DeleteSolution", mock.Anything, challengeID).Return(nil)

	err := uc.AdminDeleteSolution(context.Background(), challengeID)

	assert.NoError(t, err)
}

func TestAdminDeleteSolution_RepoError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()

	deps.challengeRepo.On("DeleteSolution", mock.Anything, challengeID).Return(errors.New("db error"))

	err := uc.AdminDeleteSolution(context.Background(), challengeID)

	assert.Error(t, err)
}

func TestAdminDeleteSolution_NonExistent_IsIdempotent(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()

	deps.challengeRepo.On("DeleteSolution", mock.Anything, challengeID).Return(nil)

	err := uc.AdminDeleteSolution(context.Background(), challengeID)

	assert.NoError(t, err)
}

func TestListSolutions_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	teamID := uuid.New()
	cid1 := uuid.New()
	cid2 := uuid.New()

	entries := []*repo.ChallengeSolutionEntry{
		{ChallengeID: cid1, ChallengeTitle: "Web 1", ChallengeCategory: "web", Content: "## WS1", Files: []*entity.File{}},
		{ChallengeID: cid2, ChallengeTitle: "Pwn 1", ChallengeCategory: "pwn", Content: "## PS1", Files: []*entity.File{}},
	}
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil)
	deps.challengeRepo.On("ListSolutions", mock.Anything, teamID).Return(entries, nil)

	result, err := uc.ListSolutions(context.Background(), teamID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Web 1", result[0].ChallengeTitle)
	assert.Equal(t, "Pwn 1", result[1].ChallengeTitle)
}

func TestListSolutions_Empty(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	teamID := uuid.New()

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil)
	deps.challengeRepo.On("ListSolutions", mock.Anything, teamID).Return([]*repo.ChallengeSolutionEntry{}, nil)

	result, err := uc.ListSolutions(context.Background(), teamID)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestListSolutions_RepoError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	teamID := uuid.New()

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil)
	deps.challengeRepo.On("ListSolutions", mock.Anything, teamID).Return(nil, errors.New("db error"))

	result, err := uc.ListSolutions(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetSolution_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solve := &entity.Solve{ID: uuid.New(), TeamID: teamID, ChallengeID: challengeID}
	solution := &repo.ChallengeSolution{ChallengeID: challengeID, Content: "## Solution", Files: []*entity.File{}}

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(solve, nil)
	deps.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(solution, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "## Solution", result.Content)
}

func TestGetSolution_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
}

func TestGetSolution_NoTeamID_Forbidden(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	result, err := uc.GetSolution(context.Background(), challengeID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, httperr.ErrNotAuthenticated))
}

func TestGetSolution_NotSolved_Forbidden(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetSolution_SolutionNotFound(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Web Chall", "web", 100, "hash")
	solve := &entity.Solve{ID: uuid.New(), TeamID: teamID, ChallengeID: challengeID}

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(solve, nil)
	deps.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	result, err := uc.GetSolution(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
