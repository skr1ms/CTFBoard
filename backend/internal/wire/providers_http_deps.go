package wire

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsController "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/sse"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func ProvideServerDeps(
	ctx context.Context,
	cfg *config.Config,
	userUC *user.UserUseCase,
	challengeUC *challenge.ChallengeUseCase,
	solveUC *competition.SolveUseCase,
	teamUC *team.TeamUseCase,
	competitionUC *competition.CompetitionUseCase,
	hintUC *challenge.HintUseCase,
	emailUC *email.EmailUseCase,
	fileUC *challenge.FileUseCase,
	awardUC *team.AwardUseCase,
	statsUC *competition.StatisticsUseCase,
	submissionUC *competition.SubmissionUseCase,
	tagUC *challenge.TagUseCase,
	topicUC *challenge.TopicUseCase,
	fieldUC *settings.FieldUseCase,
	pageUC *page.PageUseCase,
	bracketUC *competition.BracketUseCase,
	shareUC *competition.ShareUseCase,
	notifUC *notification.NotificationUseCase,
	apiTokenUC *user.APITokenUseCase,
	backupUC *backup.BackupUseCase,
	settingsUC *settings.SettingsUseCase,
	storageAdminUC usecase.StorageAdminUseCase,
	competitionParamUC *competition.CompetitionParamUseCase,
	commentUC *challenge.CommentUseCase,
	ratingUC *challenge.RatingUseCase,
	trackingUC *user.TrackingUseCase,
	oauthUC *user.OAuthUseCase,
	avatarUC *avatar.AvatarUseCase,
	appealUC *user.BanAppealUseCase,
	jwtService *jwtkit.JWTService,
	redisClient *redis.Client,
	wsCtrl *wsController.Controller,
	wsHub *wskit.Hub,
	v validator.Validator,
	runtimeInvalidator *runtimeSettingsInvalidator,
	TM repo.TransactionManager,
	l logkit.Logger,
) (*helper.ServerDeps, error) {
	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyForgot, forgotPasswordRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create forgot-password rate limiter: %w", err)
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResend, resendVerificationRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create resend-verification rate limiter: %w", err)
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResetTok, resetPasswordTokenRateLimit, resetPasswordTokenRateWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create reset-password-token rate limiter: %w", err)
	}

	rateLimitCache := restapimiddleware.NewRateLimitConfigCache(ctx, rateLimitCacheTTL)
	runtimeInvalidator.SetRateLimitCache(rateLimitCache)

	ratelimitAuditWG := &sync.WaitGroup{}

	return &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ReadUC:    challengeUC,
			SubmitUC:  challengeUC,
			AdminUC:   challengeUC,
			HintUC:    hintUC,
			FileUC:    fileUC,
			TagUC:     tagUC,
			TopicUC:   topicUC,
			CommentUC: commentUC,
			RatingUC:  ratingUC,
		},
		Team: helper.TeamDeps{
			ReadUC:  teamUC,
			SelfUC:  teamUC,
			AdminUC: teamUC,
			AwardUC: awardUC,
		},
		User: helper.UserDeps{
			UserUC:              userUC,
			EmailUC:             emailUC,
			APITokenUC:          apiTokenUC,
			TrackingUC:          trackingUC,
			OAuthUC:             oauthUC,
			AvatarUC:            avatarUC,
			AppealUC:            appealUC,
			FrontendURL:         cfg.FrontendURL,
			SecureCookies:       cfg.SecureCookies,
			RefreshCookieMaxAge: int(cfg.RefreshTTL.Seconds()),
			OAuthGitHubEnabled:  cfg.GitHub.IsConfigured(),
			OAuthGoogleEnabled:  cfg.Google.IsConfigured(),
		},
		Comp: helper.CompetitionDeps{
			CompetitionUC: competitionUC,
			SolveUC:       solveUC,
			ShareUC:       shareUC,
			StatsUC:       statsUC,
			SubmissionUC:  submissionUC,
			BracketUC:     bracketUC,
		},
		Admin: helper.AdminDeps{
			BackupUC:           backupUC,
			SettingsUC:         settingsUC,
			CompetitionParamUC: competitionParamUC,
			StorageAdminUC:     storageAdminUC,
			FieldUC:            fieldUC,
			PageUC:             pageUC,
			NotifUC:            notifUC,
		},
		Infra: helper.InfraDeps{
			JWTService:                    jwtService,
			RedisClient:                   redisClient,
			WSController:                  wsCtrl,
			SSEHandler:                    sse.NewSSEHandler(wsHub, l),
			Validator:                     v,
			Logger:                        l,
			TrustedProxyCIDRs:             cfg.TrustedProxyCIDRs,
			StructuredLogger:              cfg.StructuredLogger,
			DebugEnabled:                  cfg.DebugEnabled,
			RateLimitConfigCache:          rateLimitCache,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
			RatelimitAuditWG:              ratelimitAuditWG,
			TM:                            TM,
		},
	}, nil
}
