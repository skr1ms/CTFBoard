package load_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	backupUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	challengeUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	competitionUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	emailUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	pageUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	settingsUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	teamUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	userUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type teamBracketGetterImpl struct {
	r repo.TeamRepository
}

func (g *teamBracketGetterImpl) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	t, err := g.r.GetByID(ctx, teamID)
	if err != nil || t == nil {
		return nil, err
	}

	return t.BracketID, nil
}

type noOpMailer struct{}

func (m *noOpMailer) Send(_ context.Context, _ usecase.EmailMessage) error { return nil }

type loadTestDeps struct {
	log    logkit.Logger
	val    validator.Validator
	jwt    *jwtkit.JWTService
	crypto *crypto.CryptoService
}

type loadTestRepos struct {
	apiTokenRepo     *persistent.APITokenRepo
	SettingsRepo     *persistent.SettingsRepo
	auditLogRepo     *persistent.AuditLogRepo
	awardRepo        *persistent.AwardRepo
	backupRepo       *persistent.BackupRepo
	bracketRepo      *persistent.BracketRepo
	challengeRepo    *persistent.ChallengeRepo
	commentRepo      *persistent.CommentRepo
	compRepo         *persistent.CompetitionRepo
	configRepo       *persistent.CompetitionParamRepo
	fieldRepo        *persistent.FieldRepo
	fieldValueRepo   *persistent.FieldValueRepo
	fileRepo         *persistent.FileRepo
	hintRepo         *persistent.HintRepo
	notificationRepo *persistent.NotificationRepo
	pageRepo         *persistent.PageRepo
	solveRepo        *persistent.SolveRepo
	statsRepo        *persistent.StatisticsRepo
	submissionRepo   *persistent.SubmissionRepo
	tagRepo          *persistent.TagRepo
	teamRepo         *persistent.TeamRepo
	tokenRepo        *persistent.VerificationTokenRepo
	trackingRepo     *persistent.TrackingRepo
	tm               repo.TransactionManager
	userRepo         *persistent.UserRepo
}

type loadTestUseCases struct {
	user               *userUC.UserUseCase
	team               *teamUC.TeamUseCase
	award              *teamUC.AwardUseCase
	email              *emailUC.EmailUseCase
	challenge          *challengeUC.ChallengeUseCase
	hint               *challengeUC.HintUseCase
	file               *challengeUC.FileUseCase
	solve              *competitionUC.SolveUseCase
	competition        *competitionUC.CompetitionUseCase
	backup             *backupUC.BackupUseCase
	stats              *competitionUC.StatisticsUseCase
	settings           *settingsUC.SettingsUseCase
	ws                 *wsV1.Controller
	submissionUC       *competitionUC.SubmissionUseCase
	tagUC              *challengeUC.TagUseCase
	fieldUC            *settingsUC.FieldUseCase
	pageUC             *pageUC.PageUseCase
	bracketUC          *competitionUC.BracketUseCase
	notifUC            usecase.NotificationUseCase
	apiTokenUC         usecase.APITokenUseCase
	competitionParamUC *competitionUC.CompetitionParamUseCase
	commentUC          *challengeUC.CommentUseCase
	trackingUC         *userUC.TrackingUseCase
	SettingsRepo       repo.SettingsRepository
}

func initLoadTestDeps(redisClient *redis.Client) (*loadTestDeps, error) {
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	val, err := validator.New()
	if err != nil {
		return nil, fmt.Errorf("create validator: %w", err)
	}

	revoker := jwtkit.NewRedisRevocationStore(redisClient)

	jwtSvc, err := jwtkit.NewJWTService(jwtkit.Config{
		AccessKeys:  []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("test-access-secret-min-32-bytes!")}},
		RefreshKeys: []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("test-refresh-secret-min32-bytes!")}},
		AccessTTL:   24 * time.Hour,
		RefreshTTL:  72 * time.Hour,
		Issuer:      "loadtest-issuer",
		Revoker:     revoker,
	})
	if err != nil {
		return nil, fmt.Errorf("create jwt: %w", err)
	}

	cryptoSvc, err := crypto.NewCryptoService("1234567890123456789012345678901212345678901234567890123456789012")
	if err != nil {
		return nil, fmt.Errorf("create crypto: %w", err)
	}

	return &loadTestDeps{log: l, val: val, jwt: jwtSvc, crypto: cryptoSvc}, nil
}

func initLoadTestRepos(pool *pgxpool.Pool) *loadTestRepos {
	tm := persistent.NewTransactionManager(pool)

	return &loadTestRepos{
		userRepo:         persistent.NewUserRepo(pool),
		challengeRepo:    persistent.NewChallengeRepo(pool),
		solveRepo:        persistent.NewSolveRepo(pool),
		teamRepo:         persistent.NewTeamRepo(pool),
		compRepo:         persistent.NewCompetitionRepo(pool),
		hintRepo:         persistent.NewHintRepo(pool),
		awardRepo:        persistent.NewAwardRepo(pool),
		tm:               tm,
		tokenRepo:        persistent.NewVerificationTokenRepo(pool),
		auditLogRepo:     persistent.NewAuditLogRepo(pool),
		statsRepo:        persistent.NewStatisticsRepo(pool),
		fileRepo:         persistent.NewFileRepo(pool),
		backupRepo:       persistent.NewBackupRepo(pool),
		SettingsRepo:     persistent.NewSettingsRepo(pool),
		tagRepo:          persistent.NewTagRepo(pool),
		fieldRepo:        persistent.NewFieldRepo(pool),
		fieldValueRepo:   persistent.NewFieldValueRepo(pool),
		submissionRepo:   persistent.NewSubmissionRepo(pool),
		pageRepo:         persistent.NewPageRepo(pool),
		bracketRepo:      persistent.NewBracketRepo(pool),
		notificationRepo: persistent.NewNotificationRepo(pool),
		apiTokenRepo:     persistent.NewAPITokenRepo(pool),
		configRepo:       persistent.NewCompetitionParamRepo(pool),
		commentRepo:      persistent.NewCommentRepo(pool),
		trackingRepo:     persistent.NewTrackingRepo(pool),
	}
}
