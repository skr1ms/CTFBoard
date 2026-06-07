package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestChallengeUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		err := fn(ctx)
		if err != nil {
			return
		}
	})
	d.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.Title == "New Challenge" && c.Points == 200
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*domain.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	challenge, err := uc.Create(context.Background(), challengeCreateParams("New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false))

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "New Challenge", challenge.Title)
	assert.Equal(t, 200, challenge.Points)
	assert.NotEmpty(t, challenge.FlagHash)
}

func TestChallengeUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	expectedError := assert.AnError
	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	challenge, err := uc.Create(context.Background(), challengeCreateParams("New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false))

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := newTestChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		err := fn(ctx)
		if err != nil {
			return
		}
	})
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.ID == challengeID && c.Title == "Updated Title" && c.Points == 150
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 500, 100, 20
	ci, ma, pos := "", 0, 0
	ir, ic := false, false
	challenge, err := uc.Update(context.Background(), challengeID, challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", &ci, &ma, nil, &pos, "visible", &ir, &ic))

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "Updated Title", challenge.Title)
	assert.Equal(t, 150, challenge.Points)
}

func TestChallengeUseCase_Update_WithNewFlag(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := newTestChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		err := fn(ctx)
		if err != nil {
			return
		}
	})
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.ID == challengeID && c.FlagHash != "old_hash"
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 500, 100, 20
	ci, ma, pos := "", 0, 0
	ir, ic := false, false
	challenge, err := uc.Update(context.Background(), challengeID, challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "new_flag", &ci, &ma, nil, &pos, "visible", &ir, &ic))

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotEqual(t, "old_hash", challenge.FlagHash)
}

func TestChallengeUseCase_Update_GetByIDError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	expectedError := assert.AnError

	d.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	iv, mv, dc := 500, 100, 20
	ci, ma, pos := "", 0, 0
	ir, ic := false, false
	challenge, err := uc.Update(context.Background(), challengeID, challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", &ci, &ma, nil, &pos, "visible", &ir, &ic))

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update_UpdateError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := newTestChallenge(challengeID, "Old Title", "Web", 100, "old_hash")
	expectedError := assert.AnError

	d.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.challengeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedError)

	iv, mv, dc := 500, 100, 20
	ci, ma, pos := "", 0, 0
	ir, ic := false, false
	challenge, err := uc.Update(context.Background(), challengeID, challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", &ci, &ma, nil, &pos, "visible", &ir, &ic))

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := &domain.Challenge{ID: challengeID, Title: "ToDelete"}

	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		_ = fn(args.Get(0).(context.Context))
	})
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("Delete", mock.Anything, challengeID).Return(nil)
	d.auditLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.AuditLog) bool {
		return a.Action == "delete" && a.EntityID == challengeID.String() && a.EntityType == domain.AuditEntityChallenge
	})).Return(nil)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
}

func TestChallengeUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	expectedError := assert.AnError
	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
}

func TestChallengeUseCase_GetFlags_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	expected := &domain.ChallengeFlags{
		FlagHash:          "abc123hash",
		IsRegex:           false,
		IsCaseInsensitive: true,
	}

	d.challengeRepo.EXPECT().GetFlags(mock.Anything, challengeID).Return(expected, nil)

	result, err := uc.GetFlags(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestChallengeUseCase_GetFlags_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	expectedErr := errors.New("db error")

	d.challengeRepo.EXPECT().GetFlags(mock.Anything, challengeID).Return(nil, expectedErr)

	result, err := uc.GetFlags(context.Background(), challengeID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "GetFlags")
}

func TestChallengeUseCase_GetMissingChallengesByTeamID_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	expected := []*domain.Challenge{
		newTestChallenge(uuid.New(), "Missing Challenge", "Web", 100, "flaghash"),
	}

	d.challengeRepo.EXPECT().GetMissingChallengesByTeamID(mock.Anything, teamID).Return(expected, nil)

	result, err := uc.GetMissingChallengesByTeamID(context.Background(), teamID)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestChallengeUseCase_GetMissingChallengesByTeamID_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	expectedErr := errors.New("db error")

	d.challengeRepo.EXPECT().GetMissingChallengesByTeamID(mock.Anything, teamID).Return(nil, expectedErr)

	result, err := uc.GetMissingChallengesByTeamID(context.Background(), teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "GetMissingChallengesByTeamID")
}
