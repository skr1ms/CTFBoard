package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestChallengeUseCase_GetAll(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	teamID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		h.NewChallengeWithSolved(&entity.Challenge{
			ID:          uuid.New(),
			Title:       "Test Challenge",
			Description: "Test Description",
			Category:    "Web",
			Points:      100,
		}, true),
	}

	deps.challengeRepo.On("GetAll", mock.Anything, &teamID).Return(challenges, nil)

	result, err := uc.GetAll(context.Background(), &teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, challenges[0].Challenge.Title, result[0].Challenge.Title)
}

func TestChallengeUseCase_GetAll_NoTeamID(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challenges := []*repo.ChallengeWithSolved{
		h.NewChallengeWithSolved(&entity.Challenge{
			ID:          uuid.New(),
			Title:       "Test Challenge",
			Description: "Test Description",
			Category:    "Web",
			Points:      100,
		}, false),
	}

	deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil)).Return(challenges, nil)

	result, err := uc.GetAll(context.Background(), nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestChallengeUseCase_GetAll_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	teamID := uuid.New()
	expectedError := assert.AnError
	deps.challengeRepo.On("GetAll", mock.Anything, &teamID).Return(nil, expectedError)

	result, err := uc.GetAll(context.Background(), &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestChallengeUseCase_Create(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	deps.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Title == "New Challenge" && c.Points == 200
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*entity.Challenge)
		if !ok {
			return
		}
		c.ID = uuid.New()
	})

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "New Challenge", challenge.Title)
	assert.Equal(t, 200, challenge.Points)
	assert.NotEmpty(t, challenge.FlagHash)
}

func TestChallengeUseCase_Create_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	expectedError := assert.AnError
	deps.challengeRepo.On("Create", mock.Anything, mock.Anything).Return(expectedError)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := h.NewChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.ID == challengeID && c.Title == "Updated Title" && c.Points == 150
	})).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "Updated Title", challenge.Title)
	assert.Equal(t, 150, challenge.Points)
}

func TestChallengeUseCase_Update_WithNewFlag(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := h.NewChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.ID == challengeID && c.FlagHash != "old_hash"
	})).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "new_flag", false, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotEqual(t, "old_hash", challenge.FlagHash)
}

func TestChallengeUseCase_Update_GetByIDError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	expectedError := assert.AnError
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update_UpdateError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := h.NewChallenge(challengeID, "Old Title", "Web", 100, "old_hash")
	expectedError := assert.AnError

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.challengeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedError)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Delete(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&entity.Challenge{ID: challengeID, Title: "ToDelete"}, nil)
	deps.challengeRepo.On("Delete", mock.Anything, challengeID).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)
	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(context.Background(), nil) //nolint:errcheck // mock returns error, callback error ignored
	})
	deps.txRepo.On("CreateAuditLogTx", mock.Anything, mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == "delete" && a.EntityID == challengeID.String() && a.EntityType == entity.AuditEntityChallenge
	})).Return(nil)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
}

func TestChallengeUseCase_Delete_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	expectedError := assert.AnError
	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
}

func TestChallengeUseCase_SubmitFlag_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"
	challenge := h.NewChallenge(challengeID, "Test Challenge", "Web", 100, h.Sha256Hash(flag))

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(ctx, nil) //nolint:errcheck // mock returns error, callback error ignored
	})
	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.ChallengeID == challengeID && s.TeamID == teamID && s.UserID == userID
	})).Return(nil)
	deps.txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFlag(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Test Challenge", "Web", 100, h.Sha256Hash("flag{correct}"))

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{wrong}", userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_NoTeam(t *testing.T) {
	h := NewChallengeTestHelper(t)
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	userID := uuid.New()

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserMustBeInTeam))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_ChallengeNotFound(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, entityError.ErrChallengeNotFound)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByIDUnexpectedError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	expectedError := assert.AnError

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_AlreadySolved(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := h.Sha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}

	existingSolve := &entity.Solve{
		ID:          uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
	}

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(entityError.ErrAlreadySolved).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(ctx, nil) //nolint:errcheck // mock returns ErrAlreadySolved, callback error ignored
	})

	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrAlreadySolved))
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BeginTxError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := h.Sha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByTeamAndChallengeTxUnexpectedError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := h.Sha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(ctx, nil) //nolint:errcheck // mock returns error, callback error ignored
	})

	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CreateTxError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := h.Sha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(ctx, nil) //nolint:errcheck // mock returns error, callback error ignored
	})

	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFormat(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	regex := "^GoCTF\\{.+\\}$"
	comp := &entity.Competition{FlagRegex: &regex}
	deps.compRepo.On("Get", mock.Anything).Return(comp, nil)

	teamID := uuid.New()
	valid, err := uc.SubmitFlag(context.Background(), uuid.New(), "InvalidFlag", uuid.New(), &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrInvalidFlagFormat))
	assert.False(t, valid)
}

func TestChallengeUseCase_Create_Regex_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	encryptedFlag := "encrypted_regex"
	deps.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	deps.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*entity.Challenge)
		if !ok {
			return
		}
		c.ID = uuid.New()
	})

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
	assert.True(t, challenge.IsRegex)
}

func TestChallengeUseCase_Create_Regex_EncryptionError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	expectedError := errors.New("encryption failed")
	deps.crypto.On("Encrypt", flag).Return("", expectedError)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
	assert.Contains(t, err.Error(), "failed to encrypt regex flag")
}

func TestChallengeUseCase_Update_Regex_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Old Challenge",
		IsRegex:  false,
		FlagHash: "somehash",
	}

	flag := "^flag{new}$"
	encryptedFlag := "encrypted_new_regex"
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	deps.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
}

func TestChallengeUseCase_Update_Regex_EncryptionError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		ID:    challengeID,
		Title: "Old Challenge",
	}

	flag := "^flag{new}$"
	expectedError := errors.New("encryption failed")
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.crypto.On("Encrypt", flag).Return("", expectedError)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_SubmitFlag_Regex_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test_regex_match}"
	regexPattern := "^flag\\{test_regex_match\\}$"
	encryptedRegex := "encrypted_regex_pattern"

	challenge := &entity.Challenge{
		ID:        challengeID,
		Title:     "Regex Challenge",
		IsRegex:   true,
		FlagRegex: encryptedRegex,
		Points:    100,
	}

	deps.compRepo.On("Get", mock.Anything).Return(&entity.Competition{}, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.crypto.On("Decrypt", encryptedRegex).Return(regexPattern, nil)
	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(ctx, nil) //nolint:errcheck // mock returns error, callback error ignored
	})
	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_Regex_DecryptionError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"
	encryptedRegex := "encrypted_regex_pattern"

	challenge := &entity.Challenge{
		ID:        challengeID,
		IsRegex:   true,
		FlagRegex: encryptedRegex,
	}

	deps.compRepo.On("Get", mock.Anything).Return(&entity.Competition{}, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.crypto.On("Decrypt", encryptedRegex).Return("", errors.New("decryption failed"))

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CaseInsensitive_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "FLAG{CaSe_InSeNsItIvE}"
	normalizedFlag := "flag{case_insensitive}"
	flagHash := h.Sha256Hash(normalizedFlag)

	challenge := &entity.Challenge{
		ID:                challengeID,
		IsCaseInsensitive: true,
		FlagHash:          flagHash,
		Points:            100,
	}

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context, pgx.Tx) error)
		if !ok {
			return
		}
		_ = fn(ctx, nil) //nolint:errcheck // mock returns error, callback error ignored
	})

	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}
