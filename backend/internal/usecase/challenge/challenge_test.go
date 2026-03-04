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

func TestChallengeUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
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

	deps.challengeRepo.On("GetAll", mock.Anything, &teamID, mock.Anything).Return(challenges, nil)
	deps.tagRepo.On("GetByChallengeIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID][]*entity.Tag{}, nil)

	result, err := uc.GetAll(context.Background(), &teamID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, challenges[0].Challenge.Title, result[0].Challenge.Title)
}

func TestChallengeUseCase_GetAll_NoTeamID(t *testing.T) {
	t.Parallel()
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

	deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), mock.Anything).Return(challenges, nil)
	deps.tagRepo.On("GetByChallengeIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID][]*entity.Tag{}, nil)

	result, err := uc.GetAll(context.Background(), nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestChallengeUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	teamID := uuid.New()
	expectedError := assert.AnError
	deps.challengeRepo.On("GetAll", mock.Anything, &teamID, mock.Anything).Return(nil, expectedError)

	result, err := uc.GetAll(context.Background(), &teamID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestChallengeUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		if err := fn(ctx); err != nil {
			return
		}
	})
	deps.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Title == "New Challenge" && c.Points == 200
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*entity.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	deps.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "New Challenge", challenge.Title)
	assert.Equal(t, 200, challenge.Points)
	assert.NotEmpty(t, challenge.FlagHash)
}

func TestChallengeUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	expectedError := assert.AnError
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := h.NewChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		if err := fn(ctx); err != nil {
			return
		}
	})
	deps.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.ID == challengeID && c.Title == "Updated Title" && c.Points == 150
	})).Return(nil)
	deps.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "Updated Title", challenge.Title)
	assert.Equal(t, 150, challenge.Points)
}

func TestChallengeUseCase_Update_WithNewFlag(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := h.NewChallenge(challengeID, "Old Title", "Web", 100, "old_hash")

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		if err := fn(ctx); err != nil {
			return
		}
	})
	deps.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.ID == challengeID && c.FlagHash != "old_hash"
	})).Return(nil)
	deps.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "new_flag", false, false, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotEqual(t, "old_hash", challenge.FlagHash)
}

func TestChallengeUseCase_Update_GetByIDError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	expectedError := assert.AnError
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update_UpdateError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	existingChallenge := h.NewChallenge(challengeID, "Old Title", "Web", 100, "old_hash")
	expectedError := assert.AnError

	deps.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.challengeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedError)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false, false, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	challenge := &entity.Challenge{ID: challengeID, Title: "ToDelete"}
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(args.Get(0).(context.Context)) //nolint:errcheck
	})
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("Delete", mock.Anything, challengeID).Return(nil)
	deps.auditLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == "delete" && a.EntityID == challengeID.String() && a.EntityType == entity.AuditEntityChallenge
	})).Return(nil)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
}

func TestChallengeUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	expectedError := assert.AnError
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	err := uc.Delete(context.Background(), challengeID, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
}

func TestChallengeUseCase_SubmitFlag_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"
	challenge := h.NewChallenge(challengeID, "Test Challenge", "Web", 100, h.Sha256Hash(flag))
	team := h.NewTeam(teamID)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.solveRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.ChallengeID == challengeID && s.TeamID == teamID && s.UserID == userID
	})).Return(nil)
	deps.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFlag(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Test Challenge", "Web", 100, h.Sha256Hash("flag{correct}"))
	team := h.NewTeam(teamID)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{wrong}", userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_NoTeam(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	userID := uuid.New()

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserMustBeInTeam))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BannedTeam(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	bannedTeam := h.NewBannedTeam(teamID)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(bannedTeam, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamBanned))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	team := h.NewTeam(teamID)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByIDUnexpectedError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	expectedError := assert.AnError
	team := h.NewTeam(teamID)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_AlreadySolved(t *testing.T) {
	t.Parallel()
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
	team := h.NewTeam(teamID)

	existingSolve := &entity.Solve{
		ID:          uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
	}

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	deps.tm.On("Run", mock.Anything, mock.Anything).Return(httperr.ErrAlreadySolved).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrAlreadySolved))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BeginTxError(t *testing.T) {
	t.Parallel()
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
	team := h.NewTeam(teamID)
	expectedError := assert.AnError

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CreateTxError(t *testing.T) {
	t.Parallel()
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
	team := h.NewTeam(teamID)
	expectedError := assert.AnError

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	deps.solveRepo.On("Create", mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByTeamAndChallengeTxUnexpectedError(t *testing.T) {
	t.Parallel()
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
	team := h.NewTeam(teamID)
	expectedError := assert.AnError

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	deps.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFormat(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := &entity.Challenge{
		ID:              challengeID,
		FlagHash:        "hash",
		IsRegex:         false,
		FlagFormatRegex: nil,
	}
	team := h.NewTeam(teamID)

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	regex := "^GoCTF\\{.+\\}$"
	comp := h.NewActiveCompetition()
	comp.FlagRegex = &regex
	deps.compRepo.On("Get", mock.Anything).Return(comp, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "InvalidFlag", uuid.New(), &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrInvalidFlagFormat))
	assert.False(t, valid)
}

func TestChallengeUseCase_Create_Regex_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	encryptedFlag := "encrypted_regex"
	deps.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		if err := fn(ctx); err != nil {
			return
		}
	})
	deps.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*entity.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	deps.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
	assert.True(t, challenge.IsRegex)
}

func TestChallengeUseCase_Create_Regex_EncryptionError(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	expectedError := errors.New("encryption failed")
	deps.crypto.On("Encrypt", flag).Return("", expectedError)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
	assert.Contains(t, err.Error(), "Encrypt")
	assert.Contains(t, err.Error(), expectedError.Error())
}

func TestChallengeUseCase_Update_Regex_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

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
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		if err := fn(ctx); err != nil {
			return
		}
	})
	deps.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil)
	deps.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
}

func TestChallengeUseCase_Update_Regex_EncryptionError(t *testing.T) {
	t.Parallel()
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
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	deps.crypto.On("Encrypt", flag).Return("", expectedError)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_SubmitFlag_Regex_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCaseWithCompAndCrypto()

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
	team := h.NewTeam(teamID)

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	deps.crypto.On("Decrypt", encryptedRegex).Return(regexPattern, nil)
	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_Regex_DecryptionError(t *testing.T) {
	t.Parallel()
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
	team := h.NewTeam(teamID)

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	deps.crypto.On("Decrypt", encryptedRegex).Return("", errors.New("decryption failed"))

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CaseInsensitive_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

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
	team := h.NewTeam(teamID)

	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_GetRequirements_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	cat := "web"
	req1 := &repo.ChallengeRequirement{
		ChallengeID:    uuid.New(),
		ChallengeTitle: "Prereq One",
		Category:       &cat,
	}
	req2 := &repo.ChallengeRequirement{
		ChallengeID:    uuid.New(),
		ChallengeTitle: "Prereq Two",
		Category:       nil,
	}
	requirements := []*repo.ChallengeRequirement{req1, req2}

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&entity.Challenge{ID: challengeID}, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)

	got, err := uc.GetRequirements(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, req1.ChallengeID, got[0].ChallengeID)
	assert.Equal(t, req1.ChallengeTitle, got[0].ChallengeTitle)
	assert.Equal(t, req2.ChallengeID, got[1].ChallengeID)
	assert.Equal(t, req2.ChallengeTitle, got[1].ChallengeTitle)
}

func TestChallengeUseCase_GetRequirements_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	got, err := uc.GetRequirements(context.Background(), challengeID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
	assert.Nil(t, got)
}

func TestChallengeUseCase_SetRequirements_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{uuid.New(), uuid.New()}

	deps.tm.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn, _ := args.Get(1).(func(context.Context) error) //nolint:errcheck //nolint:errcheck // mock callback
		ctx, _ := args.Get(0).(context.Context)            //nolint:errcheck
		_ = fn(ctx)                                        //nolint:errcheck //nolint:errcheck // test only cares that Run was invoked
	}).Return(nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&entity.Challenge{ID: challengeID}, nil)
	deps.challengeRepo.On("SetRequirements", mock.Anything, challengeID, reqIDs).Return(nil)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.NoError(t, err)
}

func TestChallengeUseCase_SetRequirements_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{uuid.New()}
	deps.tm.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn, _ := args.Get(1).(func(context.Context) error) //nolint:errcheck
		ctx, _ := args.Get(0).(context.Context)            //nolint:errcheck
		_ = fn(ctx)                                        //nolint:errcheck
	}).Return(httperr.ErrChallengeNotFound)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
}

func TestChallengeUseCase_SubmitFlag_RequirementsNotMet(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	prereqID := uuid.New()
	flag := "flag{test}"
	challenge := h.NewChallenge(challengeID, "Main Challenge", "Web", 100, h.Sha256Hash(flag))
	team := h.NewTeam(teamID)
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	deps.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrRequirementsNotMet))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_RequirementsMet_Success(t *testing.T) {
	t.Parallel()
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	prereqID := uuid.New()
	flag := "flag{test}"
	challenge := h.NewChallenge(challengeID, "Main Challenge", "Web", 100, h.Sha256Hash(flag))
	team := h.NewTeam(teamID)
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	deps.compRepo.On("Get", mock.Anything).Return(h.NewActiveCompetition(), nil)
	deps.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	deps.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	deps.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{prereqID}, nil)

	deps.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(ctx) //nolint:errcheck
	})
	deps.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}
