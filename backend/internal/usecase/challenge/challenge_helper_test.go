package challenge

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
)

type ChallengeTestHelper struct {
	t    *testing.T
	deps *challengeTestDeps
}

type challengeTestDeps struct {
	challengeRepo *mocks.MockChallengeRepository
	solveRepo     *mocks.MockSolveRepository
	tm            *mocks.MockTransactionManager
	teamRepo      *mocks.MockTeamRepository
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

func NewChallengeTestHelper(t *testing.T) *ChallengeTestHelper {
	t.Helper()

	return &ChallengeTestHelper{
		t: t,
		deps: &challengeTestDeps{
			challengeRepo: mocks.NewMockChallengeRepository(t),
			solveRepo:     mocks.NewMockSolveRepository(t),
			tm:            mocks.NewMockTransactionManager(t),
			teamRepo:      mocks.NewMockTeamRepository(t),
			compRepo:      mocks.NewMockCompetitionRepository(t),
			auditLogRepo:  mocks.NewMockAuditLogRepository(t),
			crypto:        mocks.NewMockCryptoService(t),
			hintRepo:      mocks.NewMockHintRepository(t),
			awardRepo:     mocks.NewMockAwardRepository(t),
			fileRepo:      mocks.NewMockFileRepository(t),
			s3Provider:    mocks.NewMockS3Provider(t),
			commentRepo:   mocks.NewMockCommentRepository(t),
			tagRepo:       mocks.NewMockTagRepository(t),
		},
	}
}

func (h *ChallengeTestHelper) Deps() *challengeTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *ChallengeTestHelper) CreateChallengeUseCase() (*ChallengeUseCase, redismock.ClientMock) {
	h.t.Helper()
	return h.createChallengeUseCase(nil)
}

func (h *ChallengeTestHelper) CreateChallengeUseCaseWithCompAndCrypto() (*ChallengeUseCase, redismock.ClientMock) {
	h.t.Helper()
	return h.createChallengeUseCase(h.deps.crypto)
}

func (h *ChallengeTestHelper) createChallengeUseCase(cryptoSvc crypto.Service) (*ChallengeUseCase, redismock.ClientMock) {
	h.t.Helper()
	_, redis := redismock.NewClientMock()
	return NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: h.deps.challengeRepo,
		SolveRepo:     h.deps.solveRepo,
		TM:            h.deps.tm,
		CompRepo:      h.deps.compRepo,
		TeamRepo:      h.deps.teamRepo,
		AuditLogRepo:  h.deps.auditLogRepo,
		TagRepo:       h.deps.tagRepo,
		Crypto:        cryptoSvc,
	}), redis
}

func (h *ChallengeTestHelper) Sha256Hash(text string) string {
	h.t.Helper()
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func (h *ChallengeTestHelper) NewChallenge(id uuid.UUID, title, category string, points int, flagHash string) *entity.Challenge {
	h.t.Helper()
	return &entity.Challenge{
		ID:          id,
		Title:       title,
		Description: "Description",
		Category:    category,
		Points:      points,
		FlagHash:    flagHash,
	}
}

func (h *ChallengeTestHelper) NewChallengeWithSolved(challenge *entity.Challenge, solved bool) *repo.ChallengeWithSolved {
	h.t.Helper()
	return &repo.ChallengeWithSolved{
		Challenge: challenge,
		Solved:    solved,
	}
}

func (h *ChallengeTestHelper) NewTeam(id uuid.UUID) *entity.Team {
	h.t.Helper()
	return &entity.Team{
		ID:        id,
		Name:      "Test Team",
		IsBanned:  false,
		CaptainID: uuid.New(),
	}
}

func (h *ChallengeTestHelper) NewBannedTeam(id uuid.UUID) *entity.Team {
	h.t.Helper()
	team := h.NewTeam(id)
	team.IsBanned = true
	return team
}

func (h *ChallengeTestHelper) NewActiveCompetition() *entity.Competition {
	h.t.Helper()
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(24 * time.Hour)
	return &entity.Competition{
		StartTime: &start,
		EndTime:   &end,
	}
}
