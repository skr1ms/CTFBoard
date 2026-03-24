package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func TestHintUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).Return(&domain.Challenge{}, nil).Once()
	d.hintRepo.On("Create", mock.Anything, mock.MatchedBy(func(h *domain.Hint) bool {
		return h.Title == "test title" && h.Content == "test hint" && h.Cost == 50
	})).Return(nil).Run(func(args mock.Arguments) {
		h, ok := args.Get(1).(*domain.Hint)
		if !ok {
			return
		}
		h.ID = uuid.New()
	})

	hint, err := uc.Create(context.Background(), uuid.New(), "test title", "test hint", 50, 0)

	assert.NoError(t, err)
	assert.NotNil(t, hint)
	assert.Equal(t, "test title", hint.Title)
	assert.Equal(t, "test hint", hint.Content)
	assert.Equal(t, 50, hint.Cost)
}

func TestHintUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	d.challengeRepo.On("GetByID", mock.Anything, mock.Anything).Return(&domain.Challenge{}, nil).Once()
	d.hintRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	hint, err := uc.Create(context.Background(), uuid.New(), "", "test hint", 50, 0)

	assert.Error(t, err)
	assert.Nil(t, hint)
}

func TestHintUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()
	hint := &domain.Hint{ID: hintID, Content: "Secret hint", Cost: 50}
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.NoError(t, err)
	assert.Equal(t, hintID, result.ID)
	assert.Equal(t, "Secret hint", result.Content)
}

func TestHintUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, httperr.ErrHintNotFound)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, httperr.ErrHintNotFound))
}

func TestHintUseCase_GetByChallengeID_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	hint1ID := uuid.New()
	hint2ID := uuid.New()

	hints := []*domain.Hint{{ID: hint1ID, ChallengeID: challengeID, Content: "Hint 1", Cost: 10, OrderIndex: 0}, {ID: hint2ID, ChallengeID: challengeID, Content: "Hint 2", Cost: 20, OrderIndex: 1}}

	challenge := &domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(hints, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{hint1ID}, nil)

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, hint1ID, result[0].Hint.ID)
	assert.True(t, result[0].Unlocked)
	assert.Equal(t, "Hint 1", result[0].Hint.Content)
	assert.Equal(t, hint2ID, result[1].Hint.ID)
	assert.False(t, result[1].Unlocked)
	assert.Empty(t, result[1].Hint.Content)
}

func TestHintUseCase_GetByChallengeID_RepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(nil, errors.New("db error"))

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestHintUseCase_GetByChallengeID_UnlockRepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	hints := []*domain.Hint{{ID: uuid.New(), ChallengeID: challengeID}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(hints, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return(nil, errors.New("db error"))

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestHintUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()
	hint := &domain.Hint{ID: hintID, Content: "Old content", Cost: 50, OrderIndex: 0}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.hintRepo.On("Update", mock.Anything, mock.MatchedBy(func(h *domain.Hint) bool {
		return h.Title == "New title" && h.Content == "New content" && h.Cost == 100 && h.OrderIndex == 1
	})).Return(nil)

	result, err := uc.Update(context.Background(), hintID, "New title", "New content", 100, 1)

	assert.NoError(t, err)
	assert.Equal(t, "New title", result.Title)
	assert.Equal(t, "New content", result.Content)
	assert.Equal(t, 100, result.Cost)
	assert.Equal(t, 1, result.OrderIndex)
}

func TestHintUseCase_Update_NotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(nil, httperr.ErrHintNotFound)

	result, err := uc.Update(context.Background(), hintID, "", "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, httperr.ErrHintNotFound))
}

func TestHintUseCase_Update_RepoError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()
	hint := &domain.Hint{ID: hintID, Content: "Old content", Cost: 50}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.hintRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	result, err := uc.Update(context.Background(), hintID, "", "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestHintUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()

	d.hintRepo.On("Delete", mock.Anything, hintID).Return(nil)

	err := uc.Delete(context.Background(), hintID)

	assert.NoError(t, err)
}

func TestHintUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	hintID := uuid.New()

	d.hintRepo.On("Delete", mock.Anything, hintID).Return(errors.New("db error"))

	err := uc.Delete(context.Background(), hintID)

	assert.Error(t, err)
}

func TestHintUseCase_UnlockHint_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Content: "Secret hint", Cost: 50}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.hintRepo.On("CreateUnlock", mock.Anything, teamID, hintID).Return(nil)
	d.solveRepo.On("GetTeamScore", mock.Anything, teamID).Return(100, nil)
	d.awardRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Award) bool {
		return a.Value == -50 && a.TeamID == teamID
	})).Return(nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.NoError(t, err)
	assert.NotNil(t, unlocked)
	assert.Equal(t, hintID, unlocked.ID)
	assert.Equal(t, "Secret hint", unlocked.Content)
}

func TestHintUseCase_UnlockHint_FreeHint(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Content: "Free hint", Cost: 0}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.hintRepo.On("CreateUnlock", mock.Anything, teamID, hintID).Return(nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.NoError(t, err)
	assert.NotNil(t, unlocked)
	assert.Equal(t, "Free hint", unlocked.Content)
	d.awardRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	d.solveRepo.AssertNotCalled(t, "GetTeamScore", mock.Anything, mock.Anything)
}

func TestHintUseCase_UnlockHint_NotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	challengeID := uuid.New()
	hintID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, httperr.ErrHintNotFound)

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New(), uuid.New(), challengeID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrHintNotFound))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_TM_Run_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	expectedErr := errors.New("run tx")
	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedErr)

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New(), teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.Nil(t, unlocked)
	assert.Contains(t, err.Error(), "run tx")
}

func TestHintUseCase_UnlockHint_AlreadyUnlocked(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Cost: 50}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.solveRepo.On("GetTeamScore", mock.Anything, teamID).Return(200, nil)
	d.awardRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Award) bool {
		return a.TeamID == teamID && a.Value == -50
	})).Return(nil)
	d.hintRepo.On("CreateUnlock", mock.Anything, teamID, hintID).Return(httperr.ErrHintAlreadyUnlocked)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrHintAlreadyUnlocked))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_InsufficientPoints(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Cost: 100}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.solveRepo.On("GetTeamScore", mock.Anything, teamID).Return(50, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrInsufficientPoints))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_BannedUser(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Content: "Hint", Cost: 0}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: true}, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserBanned))
	assert.Nil(t, unlocked)
	d.teamRepo.AssertNotCalled(t, "Lock", mock.Anything, mock.Anything)
}

func TestHintUseCase_UnlockHint_UserNotInTeam(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	userID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Content: "Hint", Cost: 0}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &otherTeamID, IsBanned: false}, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamMemberNotFound))
	assert.Nil(t, unlocked)
	d.teamRepo.AssertNotCalled(t, "Lock", mock.Anything, mock.Anything)
}

func TestHintUseCase_UnlockHint_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, errors.New("db error"))

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New(), teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.Nil(t, unlocked)
}
