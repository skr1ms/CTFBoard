package load_test

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-wskit"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
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
)

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
