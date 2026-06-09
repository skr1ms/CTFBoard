package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/storageadmin"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

// testDeps holds logger, validator, JWT, crypto for e2e server.
type testDeps struct {
	logger    logkit.Logger
	validator validator.Validator
	jwt       *jwtkit.JWTService
	crypto    *crypto.CryptoService
}

// testRepos holds all persistent repositories and transaction manager for e2e.
type testRepos struct {
	apiTokenRepo     *persistent.APITokenRepo
	SettingsRepo     *persistent.SettingsRepo
	auditLogRepo     *persistent.AuditLogRepo
	awardRepo        *persistent.AwardRepo
	backupRepo       *persistent.BackupRepo
	bracketRepo      *persistent.BracketRepo
	challengeRepo    *persistent.ChallengeRepo
	commentRepo      *persistent.CommentRepo
	compRepo         *persistent.CompetitionRepo
	paramRepo        *persistent.CompetitionParamRepo
	fieldRepo        *persistent.FieldRepo
	fieldValueRepo   *persistent.FieldValueRepo
	fileRepo         *persistent.FileRepo
	hintRepo         *persistent.HintRepo
	notificationRepo *persistent.NotificationRepo
	pageRepo         *persistent.PageRepo
	ratingRepo       *persistent.RatingRepo
	solveRepo        *persistent.SolveRepo
	statsRepo        *persistent.StatisticsRepo
	submissionRepo   *persistent.SubmissionRepo
	tagRepo          *persistent.TagRepo
	teamRepo         *persistent.TeamRepo
	tokenRepo        *persistent.VerificationTokenRepo
	trackingRepo     *persistent.TrackingRepo
	banAppealRepo    *persistent.BanAppealRepo
	tm               repo.TransactionManager
	userRepo         *persistent.UserRepo
}

// testUseCases holds all application use cases and settings repo for e2e server.
type testUseCases struct {
	user               *user.UserUseCase
	team               *team.TeamUseCase
	award              *team.AwardUseCase
	email              *email.EmailUseCase
	challenge          *challenge.ChallengeUseCase
	hint               *challenge.HintUseCase
	file               *challenge.FileUseCase
	solve              *competition.SolveUseCase
	competition        *competition.CompetitionUseCase
	backup             *backup.BackupUseCase
	stats              *competition.StatisticsUseCase
	settings           *settings.SettingsUseCase
	ws                 *wsV1.Controller
	submissionUC       *competition.SubmissionUseCase
	tagUC              *challenge.TagUseCase
	fieldUC            *settings.FieldUseCase
	pageUC             *page.PageUseCase
	ratingUC           *challenge.RatingUseCase
	avatarUC           *avatar.AvatarUseCase
	bracketUC          *competition.BracketUseCase
	notifUC            usecase.NotificationUseCase
	apiTokenUC         usecase.APITokenUseCase
	storageAdminUC     usecase.StorageAdminUseCase
	competitionParamUC *competition.CompetitionParamUseCase
	commentUC          *challenge.CommentUseCase
	trackingUC         *user.TrackingUseCase
	appealUC           *user.BanAppealUseCase
	SettingsRepo       repo.SettingsRepository
	TM                 repo.TransactionManager
}

// startTestServer builds deps, use cases, router, starts HTTP server on random port; returns shutdown and temp dir cleanup.
func startTestServer() (func(), error) {
	deps, err := initTestDeps()
	if err != nil {
		return nil, err
	}

	useCases, tempStorageDir, _, err := initTestUseCases(deps)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	r := setupTestRouter(ctx, deps.logger, useCases, deps.validator, deps.jwt, tempStorageDir)

	ls := net.ListenConfig{}

	listener, err := ls.Listen(ctx, "tcp", ":0")
	if err != nil {
		return nil, err
	}

	testPort = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)

	srv := &http.Server{
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 100 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	baseURL := "http://localhost:" + testPort

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/v1/competition/status", http.NoBody)
		if err != nil {
			time.Sleep(50 * time.Millisecond)

			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			fmt.Printf("server shutdown: %v\n", err)
		}

		_ = os.RemoveAll(tempStorageDir)
	}, nil
}

// Deps (logger, validator, jwt, crypto).
func initTestDeps() (*testDeps, error) {
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	validatorService, err := validator.New()
	if err != nil {
		panic("e2e: failed to create validator: " + err.Error())
	}

	jwtRevoker := jwtkit.NewRedisRevocationStore(TestRedis)

	jwtService, err := jwtkit.NewJWTService(jwtkit.Config{
		AccessKeys:  []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("test-access-secret-min-32-bytes!")}},
		RefreshKeys: []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("test-refresh-secret-min32-bytes!")}},
		AccessTTL:   24 * time.Hour,
		RefreshTTL:  72 * time.Hour,
		Issuer:      "e2e-issuer",
		Revoker:     jwtRevoker,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init JWT service: %w", err)
	}

	dummyCrypto, err := crypto.NewCryptoService("1234567890123456789012345678901212345678901234567890123456789012")
	if err != nil {
		return nil, fmt.Errorf("failed to init crypto service: %w", err)
	}

	return &testDeps{
		logger:    l,
		validator: validatorService,
		jwt:       jwtService,
		crypto:    dummyCrypto,
	}, nil
}

// initTestRepos creates all persistent repos and transaction manager from TestPool.
func initTestRepos() *testRepos {
	tm := persistent.NewTransactionManager(TestPool)

	return &testRepos{
		userRepo:         persistent.NewUserRepo(TestPool),
		challengeRepo:    persistent.NewChallengeRepo(TestPool),
		solveRepo:        persistent.NewSolveRepo(TestPool),
		teamRepo:         persistent.NewTeamRepo(TestPool),
		compRepo:         persistent.NewCompetitionRepo(TestPool),
		hintRepo:         persistent.NewHintRepo(TestPool),
		awardRepo:        persistent.NewAwardRepo(TestPool),
		tm:               tm,
		tokenRepo:        persistent.NewVerificationTokenRepo(TestPool),
		auditLogRepo:     persistent.NewAuditLogRepo(TestPool),
		statsRepo:        persistent.NewStatisticsRepo(TestPool),
		fileRepo:         persistent.NewFileRepo(TestPool),
		backupRepo:       persistent.NewBackupRepo(TestPool),
		SettingsRepo:     persistent.NewSettingsRepo(TestPool),
		tagRepo:          persistent.NewTagRepo(TestPool),
		fieldRepo:        persistent.NewFieldRepo(TestPool),
		fieldValueRepo:   persistent.NewFieldValueRepo(TestPool),
		submissionRepo:   persistent.NewSubmissionRepo(TestPool),
		pageRepo:         persistent.NewPageRepo(TestPool),
		ratingRepo:       persistent.NewRatingRepo(TestPool),
		bracketRepo:      persistent.NewBracketRepo(TestPool),
		notificationRepo: persistent.NewNotificationRepo(TestPool),
		apiTokenRepo:     persistent.NewAPITokenRepo(TestPool),
		paramRepo:        persistent.NewCompetitionParamRepo(TestPool),
		commentRepo:      persistent.NewCommentRepo(TestPool),
		trackingRepo:     persistent.NewTrackingRepo(TestPool),
		banAppealRepo:    persistent.NewBanAppealRepo(TestPool),
	}
}

func initTestStorageAndHub() (string, storage.Provider, *wskit.Hub, error) {
	tempStorageDir, err := os.MkdirTemp("", "ctf-platform-e2e-storage")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp storage dir: %w", err)
	}

	fileStorage, err := storage.NewFilesystemProvider(tempStorageDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create storage provider: %w", err)
	}

	ctx := context.Background()

	hub := wskit.NewHub(
		wskit.WithRedis(TestRedis, "ctf-platform:events"),
		wskit.WithOnConnect(func(sub wskit.Subscriber) {
			c, ok := sub.(*wskit.Client)
			if !ok {
				return
			}

			data, err := json.Marshal(wskit.NewEvent(websocket.EventTypeConnected, nil))
			if err == nil {
				c.Send(data)
			}
		}),
	)
	go hub.Run(ctx)
	go hub.SubscribeToRedis(ctx)

	return tempStorageDir, fileStorage, hub, nil
}

func buildTestUseCases(deps *testDeps, repos *testRepos, fileStorage storage.Provider, hub *wskit.Hub) *testUseCases {
	fieldValidator := settings.NewFieldValidator(repos.fieldRepo)
	broadcaster := websocket.NewBroadcaster(context.Background(), hub)
	testCache := cachekit.New(TestRedis)
	scoreboardCache := cache.NewScoreboardCacheService(testCache, &teamBracketGetter{repos.teamRepo})
	competitionGuard := competition.NewGuard(repos.compRepo)
	userCacheSvc := cache.NewUserCacheService(testCache)
	teamUC := team.NewTeamUseCase(team.TeamDeps{
		TeamRepo:           repos.teamRepo,
		UserRepo:           repos.userRepo,
		SolveRepo:          repos.solveRepo,
		SubmissionRepo:     repos.submissionRepo,
		AwardRepo:          repos.awardRepo,
		CompRepo:           repos.compRepo,
		SettingsGetter:     repos.SettingsRepo,
		ChallengeRepo:      repos.challengeRepo,
		TM:                 repos.tm,
		Guard:              competitionGuard,
		ScoreboardCache:    scoreboardCache,
		UserCache:          userCacheSvc,
		HintRepo:           repos.hintRepo,
		TeamCache:          testCache,
		DefaultMaxTeamSize: 10,
	})
	emailUC := email.NewEmailUseCase(email.EmailDeps{
		UserRepo: repos.userRepo, TokenRepo: repos.tokenRepo, TM: repos.tm, Mailer: &noOpMailer{},
		VerifyTTL: 24 * time.Hour, ResetTTL: 1 * time.Hour, FrontendURL: "http://localhost:3000", Enabled: true,
	})

	userUC := user.NewUserUseCase(user.UserDeps{
		UserRepo:        repos.userRepo,
		TeamRepo:        repos.teamRepo,
		SolveRepo:       repos.solveRepo,
		SubmissionRepo:  repos.submissionRepo,
		AwardRepo:       repos.awardRepo,
		TM:              repos.tm,
		JWTService:      deps.jwt,
		FieldValidator:  fieldValidator,
		FieldValueRepo:  repos.fieldValueRepo,
		SettingsRepo:    repos.SettingsRepo,
		EmailSender:     emailUC,
		FailedLogin:     nil,
		CompRepo:        repos.compRepo,
		SoloTeamCreator: teamUC,
		UserCache:       userCacheSvc,
		ChallengeRepo:   repos.challengeRepo,
		ScoreboardCache: scoreboardCache,
		Logger:          deps.logger,
	})
	deps.jwt.SetUserRoleLookup(func(ctx context.Context, userID uuid.UUID) (string, error) {
		u, err := repos.userRepo.GetByID(ctx, userID)
		if err != nil {
			return "", fmt.Errorf("SetUserRoleLookup - GetByID: %w", err)
		}

		return string(u.Role), nil
	})
	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{
		CompetitionRepo: repos.compRepo,
		AuditLogRepo:    repos.auditLogRepo,
		TM:              repos.tm,
		Redis:           &cachekit.RedisKeyValueStore{Client: TestRedis},
		ScoreboardCache: scoreboardCache,
		Logger:          deps.logger,
	})
	competitionParamUC := competition.NewCompetitionParamUseCase(competition.CompetitionParamDeps{
		Repo:         repos.paramRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Logger:       deps.logger,
	})
	hintUC := challenge.NewHintUseCase(challenge.HintDeps{
		HintRepo: repos.hintRepo, AwardRepo: repos.awardRepo,
		TM: repos.tm, SolveRepo: repos.solveRepo, CompRepo: repos.compRepo, CompUC: compUC,
		TeamRepo: repos.teamRepo, UserRepo: repos.userRepo,
		ChallengeRepo: repos.challengeRepo, ScoreboardCache: scoreboardCache,
	})
	challengeUC := challenge.NewChallengeUseCase(challenge.ChallengeDeps{
		ChallengeRepo:   repos.challengeRepo,
		TagRepo:         repos.tagRepo,
		SolveRepo:       repos.solveRepo,
		SubmissionRepo:  repos.submissionRepo,
		TM:              repos.tm,
		CompRepo:        repos.compRepo,
		CompParamUC:     competitionParamUC,
		TeamRepo:        repos.teamRepo,
		UserRepo:        repos.userRepo,
		ScoreboardCache: scoreboardCache,
		ListCache:       testCache,
		Broadcaster:     broadcaster,
		AuditLogRepo:    repos.auditLogRepo,
		Crypto:          deps.crypto,
		FileRepo:        repos.fileRepo,
		Storage:         fileStorage,
		HintUC:          hintUC,
		SolveRecord:     competition.RecordSolveInTx,
		Logger:          deps.logger,
	})
	ratingUC := challenge.NewRatingUseCase(challenge.RatingDeps{
		ChallengeRepo: repos.challengeRepo,
		SolveRepo:     repos.solveRepo,
		RatingRepo:    repos.ratingRepo,
		UserRepo:      repos.userRepo,
		TeamRepo:      repos.teamRepo,
		TM:            repos.tm,
	})
	avatarUC := avatar.NewAvatarUseCase(avatar.AvatarDeps{
		UserRepo: repos.userRepo,
		TeamRepo: repos.teamRepo,
		Storage:  fileStorage,
		Cache:    &cachekit.RedisKeyValueStore{Client: TestRedis},
		Config:   domain.GetDefaultAvatarConfig(),
		Logger:   deps.logger,
	})
	solveUC := competition.NewSolveUseCase(competition.SolveDeps{
		SolveRepo: repos.solveRepo, ChallengeRepo: repos.challengeRepo, CompetitionRepo: repos.compRepo,
		CompetitionUC: compUC,
		UserRepo:      repos.userRepo, TeamRepo: repos.teamRepo, TM: repos.tm,
		Cache: testCache, ScoreboardCache: scoreboardCache, Broadcaster: broadcaster,
	})
	awardUC := team.NewAwardUseCase(team.AwardDeps{AwardRepo: repos.awardRepo, TeamRepo: repos.teamRepo, TM: repos.tm, ScoreboardCache: scoreboardCache, CompRepo: repos.compRepo})
	statsUC := competition.NewStatisticsUseCase(competition.StatisticsDeps{StatsRepo: repos.statsRepo, Cache: testCache})
	submissionUC := competition.NewSubmissionUseCase(competition.SubmissionDeps{
		SubmissionRepo: repos.submissionRepo,
		CompGetter:     compUC,
		TM:             repos.tm,
		SolveCreator:   challengeUC,
		SolveDeleter:   challengeUC,
	})
	tagUC := challenge.NewTagUseCase(challenge.TagDeps{TagRepo: repos.tagRepo, ChallengeRepo: repos.challengeRepo, SolveRepo: repos.solveRepo})
	fieldUC := settings.NewFieldUseCase(settings.FieldDeps{FieldRepo: repos.fieldRepo})
	pageUC := page.NewPageUseCase(page.PageDeps{PageRepo: repos.pageRepo})
	bracketUC := competition.NewBracketUseCase(competition.BracketDeps{BracketRepo: repos.bracketRepo, TM: repos.tm})
	notifUC := notification.NewNotificationUseCase(notification.NotificationDeps{
		NotifRepo:   repos.notificationRepo,
		TeamRepo:    repos.teamRepo,
		UserRepo:    repos.userRepo,
		TM:          repos.tm,
		Broadcaster: broadcaster,
	})
	apiTokenUC := user.NewAPITokenUseCase(user.APITokenDeps{Repo: repos.apiTokenRepo})
	backupUC := backup.NewBackupUseCase(backup.BackupDeps{
		CompetitionRepo: repos.compRepo, ChallengeRepo: repos.challengeRepo, TagRepo: repos.tagRepo, HintRepo: repos.hintRepo,
		PageRepo: repos.pageRepo,
		TeamRepo: repos.teamRepo, UserRepo: repos.userRepo, AwardRepo: repos.awardRepo,
		SolveRepo: repos.solveRepo, SubmissionRepo: repos.submissionRepo, FileRepo: repos.fileRepo,
		BackupRepo: repos.backupRepo, SettingsRepo: repos.SettingsRepo, AuditLogRepo: repos.auditLogRepo,
		BracketRepo: repos.bracketRepo, CommentRepo: repos.commentRepo, FieldRepo: repos.fieldRepo, FieldValueRepo: repos.fieldValueRepo,
		Storage: fileStorage, TM: repos.tm, Logger: deps.logger,
	})
	settingsUC := settings.NewSettingsUseCase(settings.SettingsDeps{
		Repo:         repos.SettingsRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Redis:        &cachekit.RedisKeyValueStore{Client: TestRedis},
		CompRepo:     repos.compRepo,
		Logger:       deps.logger,
	})
	commentUC := challenge.NewCommentUseCase(challenge.CommentDeps{
		CommentRepo:   repos.commentRepo,
		ChallengeRepo: repos.challengeRepo,
		SolveRepo:     repos.solveRepo,
		UserRepo:      repos.userRepo,
		TeamRepo:      repos.teamRepo,
		TM:            repos.tm,
	})
	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: repos.trackingRepo})
	appealUC := user.NewBanAppealUseCase(repos.banAppealRepo, repos.userRepo, repos.tm, userUC)
	ws := wsV1.NewController(hub, deps.logger, []string{"localhost:*"})
	fileUC := challenge.NewFileUseCase(challenge.FileDeps{
		FileRepo:       repos.fileRepo,
		ChallengeRepo:  repos.challengeRepo,
		SolveRepo:      repos.solveRepo,
		Storage:        fileStorage,
		Expiry:         1 * time.Hour,
		DownloadSecret: "test-download-secret",
		BaseURL:        "http://localhost:3000",
	})
	storageAdminUC := storageadmin.NewUseCase(storageadmin.Deps{
		Storage:  fileStorage,
		AuditLog: repos.auditLogRepo,
	})

	return &testUseCases{
		user: userUC, challenge: challengeUC, solve: solveUC, team: teamUC, competition: compUC,
		hint: hintUC, award: awardUC, email: emailUC, file: fileUC, stats: statsUC, backup: backupUC,
		settings: settingsUC, ws: ws, submissionUC: submissionUC, tagUC: tagUC, fieldUC: fieldUC,
		pageUC: pageUC, ratingUC: ratingUC, avatarUC: avatarUC, bracketUC: bracketUC, notifUC: notifUC, apiTokenUC: apiTokenUC,
		storageAdminUC: storageAdminUC, competitionParamUC: competitionParamUC, commentUC: commentUC, trackingUC: trackingUC, appealUC: appealUC,
		SettingsRepo: repos.SettingsRepo,
		TM:           repos.tm,
	}
}

// initTestUseCases initializes repos, storage, hub, and builds all use cases; returns temp dir path and storage provider.
func initTestUseCases(deps *testDeps) (*testUseCases, string, storage.Provider, error) {
	repos := initTestRepos()

	tempStorageDir, fileStorage, hub, err := initTestStorageAndHub()
	if err != nil {
		return nil, "", nil, err
	}

	uc := buildTestUseCases(deps, repos, fileStorage, hub)

	return uc, tempStorageDir, fileStorage, nil
}
