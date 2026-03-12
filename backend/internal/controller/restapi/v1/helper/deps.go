package helper

import (
	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type ChallengeDeps struct {
	ChallengeUC usecase.ChallengeUseCase
	HintUC      usecase.HintUseCase
	FileUC      usecase.FileUseCase
	TagUC       usecase.TagUseCase
	CommentUC   usecase.CommentUseCase
}

type TeamDeps struct {
	TeamUC  usecase.TeamUseCase
	AwardUC usecase.AwardUseCase
}

type UserDeps struct {
	UserUC        usecase.UserUseCase
	EmailUC       usecase.EmailUseCase
	APITokenUC    usecase.APITokenUseCase
	TrackingUC    usecase.TrackingUseCase
	OAuthUC       usecase.OAuthUseCase
	FrontendURL   string
	SecureCookies bool
}

type CompetitionDeps struct {
	CompetitionUC     usecase.CompetitionUseCase
	SolveUC           usecase.SolveUseCase
	StatsUC           usecase.StatisticsUseCase
	SubmissionUC      usecase.SubmissionUseCase
	SubmissionBatcher usecase.SubmissionBatcher
	BracketUC         usecase.BracketUseCase
}

type AdminDeps struct {
	BackupUC           usecase.BackupUseCase
	SettingsUC         usecase.SettingsUseCase
	CompetitionParamUC usecase.CompetitionParamUseCase
	FieldUC            usecase.FieldUseCase
	PageUC             usecase.PageUseCase
	NotifUC            usecase.NotificationUseCase
	SettingsRepo       repo.SettingsRepository
}

type InfraDeps struct {
	JWTService                    *jwt.JWTService
	RedisClient                   *redis.Client
	StorageProvider               storage.Provider
	WSController                  *wsV1.Controller
	Validator                     validator.Validator
	Logger                        logger.Logger
	TrustedProxyCIDRs             []string
	RateLimitConfigCache          *RateLimitConfigCache
	ScoreboardVisibilityCache     *middleware.ScoreboardVisibilityCache
	ForgotPasswordRateLimiter     *middleware.PerKeyRateLimiter
	ResendVerificationRateLimiter *middleware.PerKeyRateLimiter
	ResetPasswordTokenRateLimiter *middleware.PerKeyRateLimiter
}

type ServerDeps struct {
	Challenge ChallengeDeps
	Team      TeamDeps
	User      UserDeps
	Comp      CompetitionDeps
	Admin     AdminDeps
	Infra     InfraDeps
}
