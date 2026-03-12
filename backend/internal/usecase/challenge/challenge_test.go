package challenge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type challengeTestDeps struct {
	challengeRepo *mocks.MockChallengeRepository
	solveRepo     *mocks.MockSolveRepository
	tm            *mocks.MockTransactionManager
	teamRepo      *mocks.MockTeamRepository
	userRepo      *mocks.MockUserRepository
	compRepo      *mocks.MockCompetitionRepository
	auditLogRepo  *mocks.MockAuditLogRepository
	crypto        *mocks.MockCryptoService
	hintRepo      *mocks.MockHintRepository
	awardRepo     *mocks.MockAwardRepository
	fileRepo      *mocks.MockFileRepository
	s3Provider    *mocks.MockS3Provider
	commentRepo   *mocks.MockCommentRepository
	tagRepo       *mocks.MockTagRepository
}

func newChallengeTestDeps(t *testing.T) *challengeTestDeps {
	t.Helper()
	return &challengeTestDeps{
		challengeRepo: mocks.NewMockChallengeRepository(t),
		solveRepo:     mocks.NewMockSolveRepository(t),
		tm:            mocks.NewMockTransactionManager(t),
		teamRepo:      mocks.NewMockTeamRepository(t),
		userRepo:      mocks.NewMockUserRepository(t),
		compRepo:      mocks.NewMockCompetitionRepository(t),
		auditLogRepo:  mocks.NewMockAuditLogRepository(t),
		crypto:        mocks.NewMockCryptoService(t),
		hintRepo:      mocks.NewMockHintRepository(t),
		awardRepo:     mocks.NewMockAwardRepository(t),
		fileRepo:      mocks.NewMockFileRepository(t),
		s3Provider:    mocks.NewMockS3Provider(t),
		commentRepo:   mocks.NewMockCommentRepository(t),
		tagRepo:       mocks.NewMockTagRepository(t),
	}
}

func (d *challengeTestDeps) createChallengeUseCase() (*ChallengeUseCase, redismock.ClientMock) {
	_, redis := redismock.NewClientMock()
	return NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: nil,
	}), redis
}

func (d *challengeTestDeps) createChallengeUseCaseWithCompAndCrypto() (*ChallengeUseCase, redismock.ClientMock) {
	_, redis := redismock.NewClientMock()
	return NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: d.crypto,
	}), redis
}

func (d *challengeTestDeps) createFileUseCase() *FileUseCase {
	return NewFileUseCase(FileDeps{
		FileRepo: d.fileRepo, ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo,
		Storage: d.s3Provider, Expiry: time.Hour, DownloadSecret: "test-secret", BaseURL: "http://localhost:8080",
	})
}

func (d *challengeTestDeps) createHintUseCase() (*HintUseCase, redismock.ClientMock) {
	_, redis := redismock.NewClientMock()
	return NewHintUseCase(HintDeps{
		HintRepo: d.hintRepo, AwardRepo: d.awardRepo, TM: d.tm, SolveRepo: d.solveRepo,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, UserRepo: d.userRepo, ChallengeRepo: d.challengeRepo, ScoreboardCache: nil,
	}), redis
}

func (d *challengeTestDeps) createTagUseCase() *TagUseCase {
	return NewTagUseCase(TagDeps{TagRepo: d.tagRepo, ChallengeRepo: d.challengeRepo})
}

func (d *challengeTestDeps) createCommentUseCase() *CommentUseCase {
	return NewCommentUseCase(CommentDeps{CommentRepo: d.commentRepo, ChallengeRepo: d.challengeRepo})
}

func challengeTestSha256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func newTestChallenge(id uuid.UUID, title, category string, points int, flagHash string) *entity.Challenge {
	return &entity.Challenge{ID: id, Title: title, Description: "Description", Category: category, Points: points, FlagHash: flagHash}
}

func newTestChallengeWithSolved(challenge *entity.Challenge, solved bool) *repo.ChallengeWithSolved {
	return &repo.ChallengeWithSolved{Challenge: challenge, Solved: solved}
}

func newTestTeam(id uuid.UUID) *entity.Team {
	return &entity.Team{ID: id, Name: "Test Team", IsBanned: false, CaptainID: uuid.New()}
}

func newTestBannedTeam(id uuid.UUID) *entity.Team {
	team := newTestTeam(id)
	team.IsBanned = true
	return team
}

func newActiveCompetition() *entity.Competition {
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(24 * time.Hour)
	return &entity.Competition{StartTime: &start, EndTime: &end}
}

func newTestTag(name, color string) *entity.Tag {
	return &entity.Tag{ID: uuid.New(), Name: name, Color: color}
}

func newTestComment(userID, challengeID uuid.UUID, content string) *entity.Comment {
	return &entity.Comment{ID: uuid.New(), UserID: userID, ChallengeID: challengeID, Content: content}
}

func TestChallengeUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		newTestChallengeWithSolved(&entity.Challenge{
			ID: uuid.New(), Title: "Test Challenge", Description: "Test Description", Category: "Web", Points: 100,
		}, true),
	}

	d.compRepo.On("Get", mock.Anything).Return(nil, nil)
	d.challengeRepo.On("GetAll", mock.Anything, &teamID, mock.Anything).Return(challenges, nil)
	d.tagRepo.On("GetByChallengeIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID][]*entity.Tag{}, nil)

	result, err := uc.GetAll(context.Background(), &teamID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, challenges[0].Challenge.Title, result[0].Challenge.Title)
}

func TestChallengeUseCase_GetAll_NoTeamID(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challenges := []*repo.ChallengeWithSolved{
		newTestChallengeWithSolved(&entity.Challenge{
			ID:          uuid.New(),
			Title:       "Test Challenge",
			Description: "Test Description",
			Category:    "Web",
			Points:      100,
		}, false),
	}

	d.compRepo.On("Get", mock.Anything).Return(nil, nil)
	d.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), mock.Anything).Return(challenges, nil)
	d.tagRepo.On("GetByChallengeIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID][]*entity.Tag{}, nil)

	result, err := uc.GetAll(context.Background(), nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestChallengeUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	expectedError := assert.AnError
	d.compRepo.On("Get", mock.Anything).Return(nil, nil)
	d.challengeRepo.On("GetAll", mock.Anything, &teamID, mock.Anything).Return(nil, expectedError)

	result, err := uc.GetAll(context.Background(), &teamID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

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
		if err := fn(ctx); err != nil {
			return
		}
	})
	d.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Title == "New Challenge" && c.Points == 200
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*entity.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false, nil, nil)

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

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false, false, false, nil, nil)

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
		if err := fn(ctx); err != nil {
			return
		}
	})
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.ID == challengeID && c.Title == "Updated Title" && c.Points == 150
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 500, 100, 20
	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", false, false, false, nil, nil)

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
		if err := fn(ctx); err != nil {
			return
		}
	})
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.ID == challengeID && c.FlagHash != "old_hash"
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 500, 100, 20
	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "new_flag", false, false, false, nil, nil)

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
	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", false, false, false, nil, nil)

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
	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, &iv, &mv, &dc, "", false, false, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	challenge := &entity.Challenge{ID: challengeID, Title: "ToDelete"}
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}
		_ = fn(args.Get(0).(context.Context)) //nolint:errcheck
	})
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("Delete", mock.Anything, challengeID).Return(nil)
	d.auditLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == "delete" && a.EntityID == challengeID.String() && a.EntityType == entity.AuditEntityChallenge
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

func TestChallengeUseCase_SubmitFlag_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"
	challenge := newTestChallenge(challengeID, "Test Challenge", "Web", 100, challengeTestSha256Hash(flag))
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.solveRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.ChallengeID == challengeID && s.TeamID == teamID && s.UserID == userID
	})).Return(nil)
	d.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFlag(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := newTestChallenge(challengeID, "Test Challenge", "Web", 100, challengeTestSha256Hash("flag{correct}"))
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{wrong}", userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_NoTeam(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	userID := uuid.New()

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrUserMustBeInTeam))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BannedTeam(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	bannedTeam := newTestBannedTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(bannedTeam, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamBanned))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByIDUnexpectedError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	expectedError := assert.AnError
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_AlreadySolved(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)

	existingSolve := &entity.Solve{
		ID:          uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
	}

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	d.tm.On("Run", mock.Anything, mock.Anything).Return(httperr.ErrAlreadySolved).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrAlreadySolved))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BeginTxError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)
	expectedError := assert.AnError

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CreateTxError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)
	expectedError := assert.AnError

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	d.solveRepo.On("Create", mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByTeamAndChallengeTxUnexpectedError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)
	expectedError := assert.AnError

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, expectedError)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFormat(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := &entity.Challenge{
		ID:              challengeID,
		FlagHash:        "hash",
		IsRegex:         false,
		FlagFormatRegex: nil,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	regex := "^GoCTF\\{.+\\}$"
	comp := newActiveCompetition()
	comp.FlagRegex = &regex
	d.compRepo.On("Get", mock.Anything).Return(comp, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "InvalidFlag", uuid.New(), &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrInvalidFlagFormat))
	assert.False(t, valid)
}

func TestChallengeUseCase_Create_Regex_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	encryptedFlag := "encrypted_regex"
	d.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	d.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*entity.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
	assert.True(t, challenge.IsRegex)
}

func TestChallengeUseCase_Create_Regex_EncryptionError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	expectedError := errors.New("encryption failed")
	d.crypto.On("Encrypt", flag).Return("", expectedError)

	challenge, err := uc.Create(context.Background(), "Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, false, true, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
	assert.Contains(t, err.Error(), "Encrypt")
	assert.Contains(t, err.Error(), expectedError.Error())
}

func TestChallengeUseCase_Update_Regex_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		ID:       challengeID,
		Title:    "Old Challenge",
		IsRegex:  false,
		FlagHash: "somehash",
	}

	flag := "^flag{new}$"
	encryptedFlag := "encrypted_new_regex"
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.IsRegex && c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 0, 0, 0
	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, &iv, &mv, &dc, flag, false, true, false, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, encryptedFlag, challenge.FlagRegex)
}

func TestChallengeUseCase_Update_Regex_EncryptionError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		ID:    challengeID,
		Title: "Old Challenge",
	}

	flag := "^flag{new}$"
	expectedError := errors.New("encryption failed")
	d.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.crypto.On("Encrypt", flag).Return("", expectedError)

	iv, mv, dc := 0, 0, 0
	challenge, err := uc.Update(context.Background(), challengeID, "Updated", "Desc", "Crypto", 100, &iv, &mv, &dc, flag, false, true, false, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_SubmitFlag_Regex_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

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
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.crypto.On("Decrypt", encryptedRegex).Return(regexPattern, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_Regex_DecryptionError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

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
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.crypto.On("Decrypt", encryptedRegex).Return("", errors.New("decryption failed"))

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CaseInsensitive_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "FLAG{CaSe_InSeNsItIvE}"
	normalizedFlag := "flag{case_insensitive}"
	flagHash := challengeTestSha256Hash(normalizedFlag)

	challenge := &entity.Challenge{
		ID:                challengeID,
		IsCaseInsensitive: true,
		FlagHash:          flagHash,
		Points:            100,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)
	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_GetRequirements_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

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

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&entity.Challenge{ID: challengeID}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)

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
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, httperr.ErrChallengeNotFound)

	got, err := uc.GetRequirements(context.Background(), challengeID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
	assert.Nil(t, got)
}

func TestChallengeUseCase_SetRequirements_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{uuid.New(), uuid.New()}

	for _, reqID := range reqIDs {
		d.challengeRepo.On("GetByID", mock.Anything, reqID).Return(&entity.Challenge{ID: reqID}, nil)
	}
	d.challengeRepo.On("GetAllRequirementPairs", mock.Anything).Return([]*entity.ChallengeRequirementPair{}, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn, _ := args.Get(1).(func(context.Context) error) //nolint:errcheck //nolint:errcheck // mock callback
		ctx, _ := args.Get(0).(context.Context)            //nolint:errcheck
		_ = fn(ctx)                                        //nolint:errcheck //nolint:errcheck // test only cares that Run was invoked
	}).Return(nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&entity.Challenge{ID: challengeID}, nil)
	d.challengeRepo.On("SetRequirements", mock.Anything, challengeID, reqIDs).Return(nil)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.NoError(t, err)
}

func TestChallengeUseCase_SetRequirements_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{uuid.New()}
	d.challengeRepo.On("GetByID", mock.Anything, reqIDs[0]).Return(nil, httperr.ErrChallengeNotFound)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.Error(t, err)
	var httpErr *httperr.HTTPError
	assert.True(t, errors.As(err, &httpErr) && httpErr.Code == "VALIDATION_ERROR")
}

func TestChallengeUseCase_SetRequirements_Cycle(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{challengeID}

	for _, reqID := range reqIDs {
		d.challengeRepo.On("GetByID", mock.Anything, reqID).Return(&entity.Challenge{ID: reqID}, nil)
	}
	d.challengeRepo.On("GetAllRequirementPairs", mock.Anything).Return([]*entity.ChallengeRequirementPair{}, nil)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.Error(t, err)
	var httpErr *httperr.HTTPError
	assert.True(t, errors.As(err, &httpErr) && httpErr.Code == "VALIDATION_ERROR")
	assert.Contains(t, err.Error(), "cycle")
}

func TestChallengeUseCase_GetDetail_RequirementsNotMet_ReturnsNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	prereqID := uuid.New()
	challenge := newTestChallenge(challengeID, "Locked", "Web", 100, "")
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{}, nil)

	detail, err := uc.GetDetail(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrChallengeNotFound))
	assert.Nil(t, detail)
}

func TestChallengeUseCase_SubmitFlag_RequirementsNotMet(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	prereqID := uuid.New()
	flag := "flag{test}"
	challenge := newTestChallenge(challengeID, "Main Challenge", "Web", 100, challengeTestSha256Hash(flag))
	team := newTestTeam(teamID)
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrRequirementsNotMet))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_RequirementsMet_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	prereqID := uuid.New()
	flag := "flag{test}"
	challenge := newTestChallenge(challengeID, "Main Challenge", "Web", 100, challengeTestSha256Hash(flag))
	team := newTestTeam(teamID)
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{prereqID}, nil)

	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	d.solveRepo.On("GetByTeamAndChallengeForUpdate", mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.challengeRepo.On("IncrementSolveCount", mock.Anything, challengeID).Return(1, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}
