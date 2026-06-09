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
	d.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, apperr.ErrHintNotFound)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrHintNotFound)
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
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
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
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
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
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
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
	d.hintRepo.On("GetByIDForUpdate", mock.Anything, hintID).Return(nil, apperr.ErrHintNotFound)

	result, err := uc.Update(context.Background(), hintID, "", "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrHintNotFound)
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
