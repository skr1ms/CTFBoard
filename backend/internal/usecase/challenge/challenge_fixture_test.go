package challenge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	challengeMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mock"
)

func stubSolveRecord(_ context.Context, _ *domain.Solve, _ *domain.Challenge, _ repo.ChallengeRepository, _ repo.SolveRepository, _ ...scoring.DecayFunction) (int, error) {
	return 1, nil
}

func challengeCreateParams(title, description, category string, points, initialValue, minValue, decay int, flag string, isRegex bool) usecase.ChallengeCreateParams {
	return usecase.ChallengeCreateParams{
		Title:             title,
		Description:       description,
		Category:          category,
		Points:            points,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		Flag:              flag,
		State:             "visible",
		IsRegex:           isRegex,
		IsCaseInsensitive: false,
	}
}

func challengeUpdateParams(title, description, category string, points int, initialValue, minValue, decay *int, flag string, connectionInfo *string, maxAttempts *int, maxAttemptsWindow *time.Duration, position *int, state string, isRegex, isCaseInsensitive *bool) usecase.ChallengeUpdateParams {
	return usecase.ChallengeUpdateParams{
		Title:             title,
		Description:       description,
		Category:          category,
		Points:            points,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		Flag:              flag,
		ConnectionInfo:    connectionInfo,
		MaxAttempts:       maxAttempts,
		MaxAttemptsWindow: maxAttemptsWindow,
		Position:          position,
		State:             state,
		IsRegex:           isRegex,
		IsCaseInsensitive: isCaseInsensitive,
	}
}

func submitFlagParams(challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID) usecase.ChallengeSubmitParams {
	return usecase.ChallengeSubmitParams{
		ChallengeID: challengeID,
		Flag:        flag,
		UserID:      userID,
		TeamID:      teamID,
	}
}

type challengeTestDeps struct {
	challengeRepo  *challengeMock.MockChallengeRepository
	solveRepo      *challengeMock.MockSolveRepository
	submissionRepo *challengeMock.MockSubmissionRepository
	tm             *challengeMock.MockTransactionManager
	teamRepo       *challengeMock.MockTeamRepository
	userRepo       *challengeMock.MockUserRepository
	compRepo       *challengeMock.MockCompetitionRepository
	auditLogRepo   *challengeMock.MockAuditLogRepository
	crypto         *challengeMock.MockCryptoService
	hintRepo       *challengeMock.MockHintRepository
	awardRepo      *challengeMock.MockAwardRepository
	fileRepo       *challengeMock.MockFileRepository
	fileStorage    *challengeMock.MockFileStorage
	commentRepo    *challengeMock.MockCommentRepository
	tagRepo        *challengeMock.MockTagRepository
	topicRepo      *challengeMock.MockTopicRepository
}

func newChallengeTestDeps(t *testing.T) *challengeTestDeps {
	t.Helper()

	return &challengeTestDeps{
		challengeRepo:  challengeMock.NewMockChallengeRepository(t),
		solveRepo:      challengeMock.NewMockSolveRepository(t),
		submissionRepo: challengeMock.NewMockSubmissionRepository(t),
		tm:             challengeMock.NewMockTransactionManager(t),
		teamRepo:       challengeMock.NewMockTeamRepository(t),
		userRepo:       challengeMock.NewMockUserRepository(t),
		compRepo:       challengeMock.NewMockCompetitionRepository(t),
		auditLogRepo:   challengeMock.NewMockAuditLogRepository(t),
		crypto:         challengeMock.NewMockCryptoService(t),
		hintRepo:       challengeMock.NewMockHintRepository(t),
		awardRepo:      challengeMock.NewMockAwardRepository(t),
		fileRepo:       challengeMock.NewMockFileRepository(t),
		fileStorage:    challengeMock.NewMockFileStorage(t),
		commentRepo:    challengeMock.NewMockCommentRepository(t),
		tagRepo:        challengeMock.NewMockTagRepository(t),
		topicRepo:      challengeMock.NewMockTopicRepository(t),
	}
}

func (d *challengeTestDeps) createChallengeUseCase() (*ChallengeUseCase, redismock.ClientMock) {
	_, redis := redismock.NewClientMock()

	return NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, SubmissionRepo: nil, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: nil,
		SolveRecord: stubSolveRecord,
	}), redis
}

func (d *challengeTestDeps) expectUnlockHintDB() {
	d.hintRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.AnythingOfType("int64")).Return(nil).Once()
}

func (d *challengeTestDeps) createChallengeUseCaseWithSubmissionRepo() (*ChallengeUseCase, redismock.ClientMock) {
	_, redis := redismock.NewClientMock()

	return NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, SubmissionRepo: d.submissionRepo, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: nil,
		SolveRecord: stubSolveRecord,
	}), redis
}

func (d *challengeTestDeps) createChallengeUseCaseWithCompAndCrypto() (*ChallengeUseCase, redismock.ClientMock) {
	_, redis := redismock.NewClientMock()

	return NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, SubmissionRepo: nil, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: d.crypto,
		SolveRecord: stubSolveRecord,
	}), redis
}

func (d *challengeTestDeps) createFileUseCase() *FileUseCase {
	return NewFileUseCase(FileDeps{
		FileRepo: d.fileRepo, ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo,
		Storage: d.fileStorage, Expiry: time.Hour, DownloadSecret: "test-secret", BaseURL: "http://localhost:8080",
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
	return NewTagUseCase(TagDeps{TagRepo: d.tagRepo, ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo})
}

func (d *challengeTestDeps) createTopicUseCase() *TopicUseCase {
	return NewTopicUseCase(TopicDeps{TopicRepo: d.topicRepo, ChallengeRepo: d.challengeRepo, TM: d.tm})
}

func (d *challengeTestDeps) createCommentUseCase() *CommentUseCase {
	return NewCommentUseCase(CommentDeps{CommentRepo: d.commentRepo, ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo})
}

func challengeTestSha256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))

	return hex.EncodeToString(hash[:])
}

func newTestChallenge(id uuid.UUID, title, category string, points int, flagHash string) *domain.Challenge {
	return &domain.Challenge{
		ID: id, Title: title, Description: "Description", Category: category, Points: points, FlagHash: flagHash,
		State: domain.ChallengeStateVisible,
	}
}

func newTestChallengeWithSolved(challenge *domain.Challenge, solved bool) *domain.ChallengeWithSolved {
	return &domain.ChallengeWithSolved{Challenge: challenge, Solved: solved}
}

func newTestTeam(id uuid.UUID) *domain.Team {
	return &domain.Team{ID: id, Name: "Test Team", IsBanned: false, CaptainID: uuid.New()}
}

func newTestBannedTeam(id uuid.UUID) *domain.Team {
	team := newTestTeam(id)
	team.IsBanned = true

	return team
}

func newActiveCompetition() *domain.Competition {
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(24 * time.Hour)

	return &domain.Competition{StartTime: &start, EndTime: &end, Mode: domain.ModeTeamsOnly}
}

func newTestTag(name, color string) *domain.Tag {
	return &domain.Tag{ID: uuid.New(), Name: name, Color: color}
}

func newTestTopic(name string) *domain.Topic {
	return &domain.Topic{ID: uuid.New(), Name: name}
}

func newTestComment(userID, challengeID uuid.UUID, content string) *domain.Comment {
	return &domain.Comment{ID: uuid.New(), UserID: userID, ChallengeID: challengeID, Content: content}
}
