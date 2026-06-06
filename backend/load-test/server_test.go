package load_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	v1helper "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	backupUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	challengeUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	competitionUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	emailUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	notificationUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	pageUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	settingsUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	teamUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	userUC "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
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

func buildLoadTestUseCases(deps *loadTestDeps, repos *loadTestRepos, fileStorage storage.Provider, hub *wskit.Hub, redisClient *redis.Client) *loadTestUseCases {
	fieldValidator := settingsUC.NewFieldValidator(repos.fieldRepo)
	broadcaster := websocket.NewBroadcaster(context.Background(), hub)
	c := cachekit.New(redisClient)
	scoreboardCache := cache.NewScoreboardCacheService(c, &teamBracketGetterImpl{repos.teamRepo})
	guard := competitionUC.NewGuard(repos.compRepo)

	userCacheSvc := cache.NewUserCacheService(c)
	t := teamUC.NewTeamUseCase(teamUC.TeamDeps{
		TeamRepo:           repos.teamRepo,
		UserRepo:           repos.userRepo,
		SolveRepo:          repos.solveRepo,
		SubmissionRepo:     repos.submissionRepo,
		AwardRepo:          repos.awardRepo,
		CompRepo:           repos.compRepo,
		SettingsGetter:     repos.SettingsRepo,
		ChallengeRepo:      repos.challengeRepo,
		TM:                 repos.tm,
		Guard:              guard,
		ScoreboardCache:    scoreboardCache,
		UserCache:          userCacheSvc,
		HintRepo:           repos.hintRepo,
		DefaultMaxTeamSize: 10,
	})
	emailSvc := emailUC.NewEmailUseCase(emailUC.EmailDeps{
		UserRepo: repos.userRepo, TokenRepo: repos.tokenRepo, TM: repos.tm,
		Mailer:      &noOpMailer{},
		VerifyTTL:   24 * time.Hour,
		ResetTTL:    1 * time.Hour,
		FrontendURL: "http://localhost:3000",
		Enabled:     true,
		BcryptCost:  bcrypt.MinCost,
	})
	u := userUC.NewUserUseCase(userUC.UserDeps{
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
		EmailSender:     emailSvc,
		FailedLogin:     nil,
		CompRepo:        repos.compRepo,
		SoloTeamCreator: t,
		UserCache:       userCacheSvc,
		BcryptCost:      bcrypt.MinCost,
	})
	comp := competitionUC.NewCompetitionUseCase(competitionUC.CompetitionDeps{
		CompetitionRepo: repos.compRepo,
		AuditLogRepo:    repos.auditLogRepo,
		TM:              repos.tm,
		Redis:           &cachekit.RedisKeyValueStore{Client: redisClient},
		Logger:          deps.log,
	})
	competitionParam := competitionUC.NewCompetitionParamUseCase(competitionUC.CompetitionParamDeps{
		Repo:         repos.configRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Logger:       deps.log,
	})
	hint := challengeUC.NewHintUseCase(challengeUC.HintDeps{
		HintRepo:        repos.hintRepo,
		AwardRepo:       repos.awardRepo,
		TM:              repos.tm,
		SolveRepo:       repos.solveRepo,
		CompRepo:        repos.compRepo,
		CompUC:          comp,
		TeamRepo:        repos.teamRepo,
		UserRepo:        repos.userRepo,
		ChallengeRepo:   repos.challengeRepo,
		ScoreboardCache: scoreboardCache,
		Logger:          deps.log,
	})
	ch := challengeUC.NewChallengeUseCase(challengeUC.ChallengeDeps{
		ChallengeRepo:   repos.challengeRepo,
		TagRepo:         repos.tagRepo,
		SolveRepo:       repos.solveRepo,
		SubmissionRepo:  repos.submissionRepo,
		FileRepo:        repos.fileRepo,
		Storage:         fileStorage,
		HintUC:          hint,
		TM:              repos.tm,
		CompRepo:        repos.compRepo,
		CompUC:          comp,
		CompParamUC:     competitionParam,
		TeamRepo:        repos.teamRepo,
		UserRepo:        repos.userRepo,
		ScoreboardCache: scoreboardCache,
		ListCache:       c,
		Broadcaster:     broadcaster,
		AuditLogRepo:    repos.auditLogRepo,
		Crypto:          deps.crypto,
		Logger:          deps.log,
		SolveRecord:     competitionUC.RecordSolveInTx,
	})
	solve := competitionUC.NewSolveUseCase(competitionUC.SolveDeps{
		SolveRepo: repos.solveRepo, ChallengeRepo: repos.challengeRepo,
		CompetitionRepo: repos.compRepo, CompetitionUC: comp, UserRepo: repos.userRepo,
		TeamRepo: repos.teamRepo, TM: repos.tm,
		Cache: c, ScoreboardCache: scoreboardCache, ChallengeListCache: ch, Broadcaster: broadcaster,
	})
	award := teamUC.NewAwardUseCase(teamUC.AwardDeps{AwardRepo: repos.awardRepo, TeamRepo: repos.teamRepo, TM: repos.tm, ScoreboardCache: scoreboardCache, CompRepo: repos.compRepo})
	stats := competitionUC.NewStatisticsUseCase(competitionUC.StatisticsDeps{StatsRepo: repos.statsRepo, Cache: c})
	sub := competitionUC.NewSubmissionUseCase(competitionUC.SubmissionDeps{SubmissionRepo: repos.submissionRepo})
	tag := challengeUC.NewTagUseCase(challengeUC.TagDeps{TagRepo: repos.tagRepo, ChallengeRepo: repos.challengeRepo})
	field := settingsUC.NewFieldUseCase(settingsUC.FieldDeps{FieldRepo: repos.fieldRepo})
	pg := pageUC.NewPageUseCase(pageUC.PageDeps{PageRepo: repos.pageRepo})
	bracket := competitionUC.NewBracketUseCase(competitionUC.BracketDeps{BracketRepo: repos.bracketRepo, TM: repos.tm})
	notif := notificationUC.NewNotificationUseCase(notificationUC.NotificationDeps{NotifRepo: repos.notificationRepo, Broadcaster: broadcaster})
	apiToken := userUC.NewAPITokenUseCase(userUC.APITokenDeps{Repo: repos.apiTokenRepo})
	bk := backupUC.NewBackupUseCase(backupUC.BackupDeps{
		CompetitionRepo: repos.compRepo, ChallengeRepo: repos.challengeRepo,
		HintRepo: repos.hintRepo, TeamRepo: repos.teamRepo, UserRepo: repos.userRepo,
		AwardRepo: repos.awardRepo, SolveRepo: repos.solveRepo,
		SubmissionRepo: repos.submissionRepo, FileRepo: repos.fileRepo,
		BackupRepo: repos.backupRepo, SettingsRepo: repos.SettingsRepo,
		Storage: fileStorage, TM: repos.tm, Logger: deps.log,
	})
	sett := settingsUC.NewSettingsUseCase(settingsUC.SettingsDeps{
		Repo:         repos.SettingsRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Redis:        &cachekit.RedisKeyValueStore{Client: redisClient},
		CompRepo:     repos.compRepo,
	})
	comment := challengeUC.NewCommentUseCase(challengeUC.CommentDeps{CommentRepo: repos.commentRepo, ChallengeRepo: repos.challengeRepo, TM: repos.tm})
	tracking := userUC.NewTrackingUseCase(userUC.TrackingDeps{TrackingRepo: repos.trackingRepo})
	ws := wsV1.NewController(hub, deps.log, []string{"localhost"})
	fileSvc := challengeUC.NewFileUseCase(challengeUC.FileDeps{
		FileRepo:       repos.fileRepo,
		ChallengeRepo:  repos.challengeRepo,
		SolveRepo:      repos.solveRepo,
		Storage:        fileStorage,
		Expiry:         1 * time.Hour,
		DownloadSecret: "test-download-secret",
		BaseURL:        "http://localhost:3000",
	})

	return &loadTestUseCases{
		user: u, challenge: ch, solve: solve, team: t, competition: comp,
		hint: hint, award: award, email: emailSvc, file: fileSvc, stats: stats,
		backup: bk, settings: sett, ws: ws, submissionUC: sub, tagUC: tag,
		fieldUC: field, pageUC: pg, bracketUC: bracket, notifUC: notif,
		apiTokenUC: apiToken, competitionParamUC: competitionParam, commentUC: comment,
		trackingUC: tracking, SettingsRepo: repos.SettingsRepo,
	}
}

func buildLoadTestRouter(ctx context.Context, l logkit.Logger, uc *loadTestUseCases, val validator.Validator, jwtSvc *jwtkit.JWTService, storageDir string, redisClient *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	clientIP, err := kitMiddleware.ClientIP(nil)
	if err != nil {
		panic(err)
	}

	r.Use(kitMiddleware.RequestID(), clientIP, kitMiddleware.Recoverer(l))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/ws") {
				next.ServeHTTP(w, r)

				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, "lt:forgot", 100000, 24*time.Hour)
	if err != nil {
		panic("load-test: failed to create forgot-password rate limiter: " + err.Error())
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, "lt:resend", 100000, 24*time.Hour)
	if err != nil {
		panic("load-test: failed to create resend-verification rate limiter: " + err.Error())
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, "lt:reset-token", 100000, time.Minute)
	if err != nil {
		panic("load-test: failed to create reset-password-token rate limiter: " + err.Error())
	}

	deps := &v1helper.ServerDeps{
		Challenge: v1helper.ChallengeDeps{
			ChallengeUC: uc.challenge, HintUC: uc.hint, FileUC: uc.file,
			TagUC: uc.tagUC, CommentUC: uc.commentUC,
		},
		Team:  v1helper.TeamDeps{TeamUC: uc.team, AwardUC: uc.award},
		User:  v1helper.UserDeps{UserUC: uc.user, EmailUC: uc.email, APITokenUC: uc.apiTokenUC, TrackingUC: uc.trackingUC},
		Comp:  v1helper.CompetitionDeps{CompetitionUC: uc.competition, SolveUC: uc.solve, StatsUC: uc.stats, SubmissionUC: uc.submissionUC, BracketUC: uc.bracketUC},
		Admin: v1helper.AdminDeps{BackupUC: uc.backup, SettingsUC: uc.settings, CompetitionParamUC: uc.competitionParamUC, FieldUC: uc.fieldUC, PageUC: uc.pageUC, NotifUC: uc.notifUC},
		Infra: v1helper.InfraDeps{
			JWTService:                    jwtSvc,
			RedisClient:                   redisClient,
			WSController:                  uc.ws,
			Validator:                     val,
			Logger:                        l,
			TrustedProxyCIDRs:             nil,
			StructuredLogger:              false,
			DebugEnabled:                  false,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
		},
	}

	r.Route("/api/v1", func(apiRouter chi.Router) {
		rateLimitCache := restapimiddleware.NewRateLimitConfigCache(context.Background(), 30*time.Second)
		v1.NewRouter(ctx, apiRouter, deps, false, rateLimitCache)

		apiRouter.Get("/files/download/*", func(w http.ResponseWriter, r *http.Request) {
			fs := http.StripPrefix("/api/v1/files/download/", http.FileServer(http.Dir(storageDir)))
			fs.ServeHTTP(w, r)
		})
	})

	return r
}

func startLoadTestServer(pool *pgxpool.Pool, redisClient *redis.Client) (baseURL string, shutdown func(), err error) {
	deps, err := initLoadTestDeps(redisClient)
	if err != nil {
		return "", nil, err
	}

	repos := initLoadTestRepos(pool)

	storageDir, err := os.MkdirTemp("", "lt-storage-")
	if err != nil {
		return "", nil, fmt.Errorf("create storage dir: %w", err)
	}

	fileStorage, err := storage.NewFilesystemProvider(storageDir)
	if err != nil {
		_ = os.RemoveAll(storageDir)

		return "", nil, fmt.Errorf("create storage: %w", err)
	}

	ctx := context.Background()

	hub := wskit.NewHub(
		wskit.WithRedis(redisClient, "lt:events"),
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

	uc := buildLoadTestUseCases(deps, repos, fileStorage, hub, redisClient)
	r := buildLoadTestRouter(ctx, deps.log, uc, deps.val, deps.jwt, storageDir, redisClient)

	ls := net.ListenConfig{}

	listener, err := ls.Listen(ctx, "tcp", ":0")
	if err != nil {
		_ = os.RemoveAll(storageDir)

		return "", nil, fmt.Errorf("listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	srv := &http.Server{
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 200 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("[load-test] server error: %v\n", err)
		}
	}()

	baseURL = fmt.Sprintf("http://localhost:%d", port)

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

	return baseURL, func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		serr := srv.Shutdown(shutCtx)
		if serr != nil {
			fmt.Printf("[load-test] shutdown: %v\n", serr)
		}

		_ = os.RemoveAll(storageDir)
	}, nil
}
