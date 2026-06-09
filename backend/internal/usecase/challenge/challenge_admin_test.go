package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
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

func TestChallengeUseCase_Create_WithMetadata(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	nextID := uuid.New()
	d.challengeRepo.On("GetByID", mock.Anything, nextID).Return(&domain.Challenge{ID: nextID}, nil).Once()
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		_ = fn(ctx)
	})
	d.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.Attribution == "Author" && c.NextChallengeID != nil && *c.NextChallengeID == nextID
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*domain.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	params := challengeCreateParams("New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false)
	params.Attribution = "Author"
	params.NextChallengeID = &nextID
	challenge, err := uc.Create(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.Equal(t, "Author", challenge.Attribution)
	require.NotNil(t, challenge.NextChallengeID)
	assert.Equal(t, nextID, *challenge.NextChallengeID)
}

func TestChallengeUseCase_Create_UnknownNextIDValidation(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	nextID := uuid.New()
	d.challengeRepo.On("GetByID", mock.Anything, nextID).Return(nil, apperr.ErrChallengeNotFound).Once()

	params := challengeCreateParams("New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false)
	params.NextChallengeID = &nextID
	challenge, err := uc.Create(context.Background(), params)

	assert.Error(t, err)
	assert.Nil(t, challenge)
	assert.Contains(t, err.Error(), "next_id references unknown challenge")

	var ve *apperr.ValidationError
	assert.True(t, errors.As(err, &ve))
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

func TestChallengeUseCase_Update_FlagFormatRegexPatchSemantics(t *testing.T) {
	t.Parallel()

	oldPattern := `^CTF\{old\}$`
	newPattern := `^CTF\{new\}$`

	tests := []struct {
		name      string
		set       bool
		value     *string
		wantValue *string
	}{
		{
			name:      "absent preserves existing",
			wantValue: &oldPattern,
		},
		{
			name:      "value updates existing",
			set:       true,
			value:     &newPattern,
			wantValue: &newPattern,
		},
		{
			name: "null clears existing",
			set:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := newChallengeTestDeps(t)
			uc, _ := d.createChallengeUseCase()

			challengeID := uuid.New()
			existingPattern := `^CTF\{old\}$`
			existingChallenge := newTestChallenge(challengeID, "Old Title", "Web", 100, "old_hash")
			existingChallenge.InitialValue = 100
			existingChallenge.MinValue = 100
			existingChallenge.FlagFormatRegex = &existingPattern

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

				_ = fn(ctx)
			})
			d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
				if tt.wantValue == nil {
					return c.FlagFormatRegex == nil
				}

				return c.FlagFormatRegex != nil && *c.FlagFormatRegex == *tt.wantValue
			})).Return(nil)
			d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			iv, mv := 100, 100
			ir, ic := false, false
			params := challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, nil, "", nil, nil, nil, nil, "visible", &ir, &ic)
			params.FlagFormatRegexSet = tt.set
			params.FlagFormatRegex = tt.value

			challenge, err := uc.Update(context.Background(), challengeID, params)
			require.NoError(t, err)

			if tt.wantValue == nil {
				assert.Nil(t, challenge.FlagFormatRegex)

				return
			}

			require.NotNil(t, challenge.FlagFormatRegex)
			assert.Equal(t, *tt.wantValue, *challenge.FlagFormatRegex)
		})
	}
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

func TestChallengeUseCase_AdminCreateSolve_UserWasInBannedTeamRejected(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	userID, teamID, challengeID := uuid.New(), uuid.New(), uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).
		Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID}, nil).Once()
	d.challengeRepo.EXPECT().GetByIDForUpdate(mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID, Points: 100}, nil).Once()
	d.userRepo.EXPECT().
		GetByID(mock.Anything, userID).
		Return(&domain.User{ID: userID, TeamID: &teamID, WasInBannedTeam: true, Role: domain.RoleUser}, nil).
		Once()

	uc := NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo,
		SolveRepo:     d.solveRepo,
		TM:            d.tm,
		TeamRepo:      d.teamRepo,
		UserRepo:      d.userRepo,
		SolveRecord:   stubSolveRecord,
	})

	err := uc.AdminCreateSolve(ctx, userID, teamID, challengeID, true)

	assert.ErrorIs(t, err, apperr.ErrUserWasInBannedTeam)
}

func TestChallengeUseCase_Update_NextIDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		nextID       uuid.UUID
		expectLookup bool
		lookupErr    error
		wantErrorIs  error
		wantValidate bool
		wantContains string
	}{
		{
			name:         "self reference",
			nextID:       uuid.Nil,
			wantValidate: true,
			wantContains: "next_id cannot reference the same challenge",
		},
		{
			name:         "missing target",
			nextID:       uuid.New(),
			expectLookup: true,
			lookupErr:    apperr.ErrChallengeNotFound,
			wantValidate: true,
			wantContains: "next_id references unknown challenge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := newChallengeTestDeps(t)
			uc, _ := d.createChallengeUseCase()
			challengeID := uuid.New()
			nextID := tt.nextID

			if nextID == uuid.Nil {
				nextID = challengeID
			}

			existingChallenge := newTestChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

			d.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil).Once()

			if tt.expectLookup {
				d.challengeRepo.On("GetByID", mock.Anything, nextID).Return(nil, tt.lookupErr).Once()
			}

			iv, mv, dc := 500, 100, 20
			ir, ic := false, false
			params := challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", nil, nil, nil, nil, "visible", &ir, &ic)
			params.NextChallengeSet = true
			params.NextChallengeID = &nextID
			challenge, err := uc.Update(context.Background(), challengeID, params)

			assert.Error(t, err)
			assert.Nil(t, challenge)

			if tt.wantErrorIs != nil {
				assert.ErrorIs(t, err, tt.wantErrorIs)
			}

			if tt.wantValidate {
				var ve *apperr.ValidationError
				assert.True(t, errors.As(err, &ve))
			}

			if tt.wantContains != "" {
				assert.Contains(t, err.Error(), tt.wantContains)
			}
		})
	}
}

func TestChallengeUseCase_Update_ClearsNextID(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	nextID := uuid.New()
	existingChallenge := newTestChallenge(challengeID, "Old Title", "Web", 100, "old_hash")
	existingChallenge.NextChallengeID = &nextID

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.ID == challengeID && c.NextChallengeID == nil
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 500, 100, 20
	ir, ic := false, false
	params := challengeUpdateParams("Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", nil, nil, nil, nil, "visible", &ir, &ic)
	params.NextChallengeSet = true
	challenge, err := uc.Update(context.Background(), challengeID, params)

	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.Nil(t, challenge.NextChallengeID)
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
