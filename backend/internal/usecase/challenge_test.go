package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func sha256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// GetAll Tests

func TestChallengeUseCase_GetAll(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	teamID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		{
			Challenge: &entity.Challenge{
				Id:          uuid.New(),
				Title:       "Test Challenge",
				Description: "Test Description",
				Category:    "Web",
				Points:      100,
			},
			Solved: true,
		},
	}

	challengeRepo.On("GetAll", mock.Anything, &teamID).Return(challenges, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	result, err := uc.GetAll(context.Background(), &teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, challenges[0].Challenge.Title, result[0].Challenge.Title)
}

func TestChallengeUseCase_GetAll_NoTeamId(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challenges := []*repo.ChallengeWithSolved{
		{
			Challenge: &entity.Challenge{
				Id:          uuid.New(),
				Title:       "Test Challenge",
				Description: "Test Description",
				Category:    "Web",
				Points:      100,
			},
			Solved: false,
		},
	}

	challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil)).Return(challenges, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	result, err := uc.GetAll(context.Background(), nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestChallengeUseCase_GetAll_Error(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	teamID := uuid.New()
	expectedError := assert.AnError

	challengeRepo.On("GetAll", mock.Anything, &teamID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	result, err := uc.GetAll(context.Background(), &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// Create Tests

func TestChallengeUseCase_Create(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Title == "New Challenge" && c.Points == 200
	})).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(1).(*entity.Challenge)
		c.Id = uuid.New()
	})

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "New Challenge", challenge.Title)
	assert.Equal(t, 200, challenge.Points)
	assert.NotEmpty(t, challenge.FlagHash)
}

func TestChallengeUseCase_Create_Error(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	expectedError := assert.AnError
	challengeRepo.On("Create", mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

// Update Tests

func TestChallengeUseCase_Update(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:          challengeID,
		Title:       "Old Title",
		Description: "Old Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "old_hash",
		IsHidden:    false,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Id == challengeID && c.Title == "Updated Title" && c.Points == 150
	})).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "Updated Title", challenge.Title)
	assert.Equal(t, 150, challenge.Points)
}

func TestChallengeUseCase_Update_WithNewFlag(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:          challengeID,
		Title:       "Old Title",
		Description: "Old Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "old_hash",
		IsHidden:    false,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Id == challengeID && c.FlagHash != "old_hash"
	})).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "new_flag", false, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotEqual(t, "old_hash", challenge.FlagHash)
}

func TestChallengeUseCase_Update_GetByIDError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update_UpdateError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:          challengeID,
		Title:       "Old Title",
		Description: "Old Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "old_hash",
		IsHidden:    false,
	}
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	challengeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

// Delete Tests

func TestChallengeUseCase_Delete(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	challengeID := uuid.New()
	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&entity.Challenge{Id: challengeID, Title: "ToDelete"}, nil)
	challengeRepo.On("Delete", mock.Anything, challengeID).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)
	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, pgx.Tx) error)
		_ = fn(context.Background(), nil)
	})

	txRepo.On("CreateAuditLogTx", mock.Anything, mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == "delete" && a.EntityId == challengeID.String() && a.EntityType == entity.AuditEntityChallenge
	})).Return(nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
}

func TestChallengeUseCase_Delete_Error(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	expectedError := assert.AnError
	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
}

// SubmitFlag Tests

func TestChallengeUseCase_SubmitFlag_Success(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.ChallengeId == challengeID && s.TeamId == teamID && s.UserId == userID
	})).Return(nil)
	txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFlag(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()

	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: sha256Hash("flag{correct}"),
		Points:   100,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{wrong}", userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_NoTeam(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	userID := uuid.New()

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserMustBeInTeam))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_ChallengeNotFound(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, entityError.ErrChallengeNotFound)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByIDUnexpectedError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_AlreadySolved(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}

	existingSolve := &entity.Solve{
		Id:          uuid.New(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrAlreadySolved))
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BeginTxError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByTeamAndChallengeTxUnexpectedError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CreateTxError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFormat(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	compRepo := mocks.NewMockCompetitionRepository(t)
	db, _ := redismock.NewClientMock()

	regex := "^GoCTF\\{.+\\}$"
	comp := &entity.Competition{FlagRegex: &regex}

	compRepo.On("Get", mock.Anything).Return(comp, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, compRepo, db, nil, nil, nil)

	teamID := uuid.New()
	valid, err := uc.SubmitFlag(context.Background(), uuid.New(), "InvalidFlag", uuid.New(), &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrInvalidFlagFormat))
	assert.False(t, valid)
}

func TestChallengeUseCase_Create_Regex_Success(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()
	cryptoService := mocks.NewMockCryptoService(t)

	flag := "^flag{test}$"
	encryptedFlag := "encrypted_regex"

	cryptoService.On("Encrypt", flag).Return(encryptedFlag, nil)

	challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(1).(*entity.Challenge)
		c.Id = uuid.New()
	})

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, cryptoService)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
	assert.True(t, challenge.IsRegex)
}

func TestChallengeUseCase_Create_Regex_EncryptionError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()
	cryptoService := mocks.NewMockCryptoService(t)

	flag := "^flag{test}$"
	expectedError := errors.New("encryption failed")

	cryptoService.On("Encrypt", flag).Return("", expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, cryptoService)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
	assert.Contains(t, err.Error(), "failed to encrypt regex flag")
}

func TestChallengeUseCase_Update_Regex_Success(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()
	cryptoService := mocks.NewMockCryptoService(t)

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Old Challenge",
		IsRegex:  false,
		FlagHash: "somehash",
	}

	flag := "^flag{new}$"
	encryptedFlag := "encrypted_new_regex"

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	cryptoService.On("Encrypt", flag).Return(encryptedFlag, nil)

	challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil)

	redisClient.ExpectDel("scoreboard").SetVal(1)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, cryptoService)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
}

func TestChallengeUseCase_Update_Regex_EncryptionError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()
	cryptoService := mocks.NewMockCryptoService(t)

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:    challengeID,
		Title: "Old Challenge",
	}

	flag := "^flag{new}$"
	expectedError := errors.New("encryption failed")

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	cryptoService.On("Encrypt", flag).Return("", expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, cryptoService)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_SubmitFlag_Regex_Success(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()
	cryptoService := mocks.NewMockCryptoService(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test_regex_match}"
	regexPattern := "^flag\\{test_regex_match\\}$"
	encryptedRegex := "encrypted_regex_pattern"

	challenge := &entity.Challenge{
		Id:        challengeID,
		Title:     "Regex Challenge",
		IsRegex:   true,
		FlagRegex: encryptedRegex,
		Points:    100,
	}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	cryptoService.On("Decrypt", encryptedRegex).Return(regexPattern, nil)

	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, cryptoService)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_Regex_DecryptionError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()
	cryptoService := mocks.NewMockCryptoService(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"
	encryptedRegex := "encrypted_regex_pattern"

	challenge := &entity.Challenge{
		Id:        challengeID,
		IsRegex:   true,
		FlagRegex: encryptedRegex,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	cryptoService.On("Decrypt", encryptedRegex).Return("", errors.New("decryption failed"))

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, cryptoService)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CaseInsensitive_Success(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "FLAG{CaSe_InSeNsItIvE}"
	normalizedFlag := "flag{case_insensitive}"
	flagHash := sha256Hash(normalizedFlag)

	challenge := &entity.Challenge{
		Id:                challengeID,
		IsCaseInsensitive: true,
		FlagHash:          flagHash,
		Points:            100,
	}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.ExpectDel("scoreboard").SetVal(1)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, nil, db, nil, nil, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}
