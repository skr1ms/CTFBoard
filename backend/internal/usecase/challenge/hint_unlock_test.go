package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestHintUseCase_UnlockHint_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()
	d.expectUnlockHintDB()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Content: "Secret hint", Cost: 50}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.hintRepo.On("CreateUnlock", mock.Anything, teamID, hintID).Return(nil)
	d.solveRepo.On("GetTeamScore", mock.Anything, teamID).Return(100, nil)
	d.awardRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Award) bool {
		return a.Value == -50 && a.TeamID == teamID
	})).Return(nil)
	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)

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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.hintRepo.On("CreateUnlock", mock.Anything, teamID, hintID).Return(nil)
	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)

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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(nil, apperr.ErrHintNotFound)

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New(), uuid.New(), challengeID, hintID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrHintNotFound)
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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{hintID}, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrHintAlreadyUnlocked)
	assert.Nil(t, unlocked)
	d.solveRepo.AssertNotCalled(t, "GetTeamScore", mock.Anything, mock.Anything)
	d.awardRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	d.hintRepo.AssertNotCalled(t, "CreateUnlock", mock.Anything, mock.Anything, mock.Anything)
}

func TestHintUseCase_UnlockHint_InsufficientPoints(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createHintUseCase()
	d.expectUnlockHintDB()

	userID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	hintID := uuid.New()

	hint := &domain.Hint{ID: hintID, ChallengeID: challengeID, Cost: 100}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: false}, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*domain.Hint{hint}, nil)
	d.hintRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{}, nil)
	d.solveRepo.On("GetTeamScore", mock.Anything, teamID).Return(50, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrInsufficientPoints)
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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &teamID, IsBanned: true}, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrUserBanned)
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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(hint, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, State: domain.ChallengeStateVisible}, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.userRepo.On("Lock", mock.Anything, userID).Return(nil)
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, TeamID: &otherTeamID, IsBanned: false}, nil)

	unlocked, err := uc.UnlockHint(context.Background(), userID, teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamMemberNotFound)
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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(nil, errors.New("db error"))

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New(), teamID, challengeID, hintID)

	assert.Error(t, err)
	assert.Nil(t, unlocked)
}
