package wire

import (
	"context"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/loginlockout"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
)

func ProvideBanAppealUseCase(
	appealRepo repo.BanAppealRepository,
	userRepo repo.UserRepository,
	tm repo.TransactionManager,
	userUC *user.UserUseCase,
) *user.BanAppealUseCase {
	return user.NewBanAppealUseCase(appealRepo, userRepo, tm, userUC)
}

func ProvideUserUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	submissionRepo repo.SubmissionRepository,
	awardRepo repo.AwardRepository,
	hintRepo repo.HintRepository,
	TM repo.TransactionManager,
	jwtService *jwtkit.JWTService,
	fieldValidator *settings.FieldValidator,
	fieldRepo repo.FieldRepository,
	fieldValueRepo repo.FieldValueRepository,
	settingsRepo repo.SettingsRepository,
	emailUC *email.EmailUseCase,
	apiTokenUC *user.APITokenUseCase,
	failedLoginTracker *loginlockout.Tracker,
	compRepo repo.CompetitionRepository,
	soloTeamCreator user.SoloTeamCreator,
	notificationUC *notification.NotificationUseCase,
	userCacheSvc *cache.UserCacheService,
	scoreboardCache *cache.ScoreboardCacheService,
	challengeListCache cacheutil.ChallengeListCacheInvalidator,
	sharedCache *cachekit.Cache,
	compParamUC *competition.CompetitionParamUseCase,
	l logkit.Logger,
) *user.UserUseCase {
	var statsInvalidator cacheutil.StatisticsCacheInvalidator

	if sharedCache != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: sharedCache}
	}

	return user.NewUserUseCase(user.UserDeps{
		UserRepo: userRepo, TeamRepo: teamRepo, SolveRepo: solveRepo,
		ChallengeRepo:  challengeRepo,
		SubmissionRepo: submissionRepo, AwardRepo: awardRepo, HintRepo: hintRepo,
		TM: TM, JWTService: jwtService,
		FieldValidator: fieldValidator, FieldRepo: fieldRepo, FieldValueRepo: fieldValueRepo,
		SettingsRepo: settingsRepo, EmailSender: emailUC, APITokenRevoker: apiTokenUC, FailedLogin: failedLoginTracker,
		CompRepo: compRepo, SoloTeamCreator: soloTeamCreator,
		PersonalNotificationSender: notificationUC,
		UserCache:                  userCacheSvc,
		ScoreboardCache:            scoreboardCache,
		StatsCache:                 statsInvalidator,
		ChallengeListCache:         challengeListCache,
		TeamCache:                  sharedCache,
		CompParamUC:                compParamUC,
		Logger:                     l,
	})
}

func ProvideTrackingUseCase(trackingRepo repo.TrackingRepository, c *cachekit.Cache, l logkit.Logger) *user.TrackingUseCase {
	var statsInvalidator competition.StatisticsCacheInvalidator

	if c != nil {
		statsInvalidator = &competition.FunnelStatsCacheInvalidatorImpl{Cache: c}
	}

	return user.NewTrackingUseCase(user.TrackingDeps{
		TrackingRepo:          trackingRepo,
		StatsCacheInvalidator: statsInvalidator,
		Logger:                l,
	})
}

func ProvideAPITokenUseCase(apiTokenRepo repo.APITokenRepository) *user.APITokenUseCase {
	return user.NewAPITokenUseCase(user.APITokenDeps{Repo: apiTokenRepo})
}

type emailMailerAdapter struct {
	mailer mailer.Mailer
}

func (a emailMailerAdapter) Send(ctx context.Context, msg email.Message) error {
	return a.mailer.Send(ctx, mailer.Message{
		To:      msg.To,
		Subject: msg.Subject,
		Body:    msg.Body,
		IsHTML:  msg.IsHTML,
	})
}

func ProvideEmailUseCase(
	userRepo email.UserRepository,
	tokenRepo email.VerificationTokenRepository,
	TM email.TransactionManager,
	mailerSvc mailer.Mailer,
	cfg *config.Config,
	competitionParamUC *competition.CompetitionParamUseCase,
	apiTokenUC *user.APITokenUseCase,
	jwtService *jwtkit.JWTService,
	l logkit.Logger,
) *email.EmailUseCase {
	return email.NewEmailUseCase(email.EmailDeps{
		UserRepo: userRepo, TokenRepo: tokenRepo, TM: TM, Mailer: emailMailerAdapter{mailer: mailerSvc},
		ConfigUC:        competitionParamUC,
		JWTRevoker:      jwtService,
		APITokenRevoker: apiTokenUC,
		VerifyTTL:       cfg.VerifyTTL, ResetTTL: cfg.ResetTTL, FrontendURL: cfg.FrontendURL, Enabled: cfg.Enabled,
		Logger: l,
	})
}
