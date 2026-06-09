package helper

import (
	"net/http"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type SetupUseCase = usecase.SetupUseCase

type ChallengeDeps struct {
	ReadUC    usecase.ChallengeReadUseCase
	SubmitUC  usecase.ChallengeSubmitUseCase
	AdminUC   usecase.ChallengeAdminUseCase
	HintUC    usecase.HintUseCase
	FileUC    usecase.FileUseCase
	TagUC     usecase.TagUseCase
	TopicUC   usecase.TopicUseCase
	CommentUC usecase.CommentUseCase
	RatingUC  usecase.RatingUseCase
}

type TeamDeps struct {
	ReadUC  usecase.TeamReadUseCase
	SelfUC  usecase.TeamSelfUseCase
	AdminUC usecase.TeamAdminUseCase
	AwardUC usecase.AwardUseCase
}

type UserDeps struct {
	UserUC              usecase.UserUseCase
	EmailUC             usecase.EmailUseCase
	APITokenUC          usecase.APITokenUseCase
	TrackingUC          usecase.TrackingUseCase
	OAuthUC             usecase.OAuthUseCase
	AvatarUC            usecase.AvatarUseCase
	AppealUC            usecase.BanAppealUseCase
	FrontendURL         string
	SecureCookies       bool
	RefreshCookieMaxAge int
	OAuthGitHubEnabled  bool
	OAuthGoogleEnabled  bool
}

type CompetitionDeps struct {
	CompetitionUC usecase.CompetitionUseCase
	SolveUC       usecase.SolveUseCase
	ShareUC       usecase.ShareUseCase
	StatsUC       usecase.StatisticsUseCase
	SubmissionUC  usecase.SubmissionUseCase
	BracketUC     usecase.BracketUseCase
}

type AdminDeps struct {
	BackupUC           usecase.BackupUseCase
	SettingsUC         usecase.SettingsUseCase
	CompetitionParamUC usecase.CompetitionParamUseCase
	StorageAdminUC     usecase.StorageAdminUseCase
	FieldUC            usecase.FieldUseCase
	PageUC             usecase.PageUseCase
	NotifUC            usecase.NotificationUseCase
}

type InfraDeps struct {
	JWTService                    *jwtkit.JWTService
	RedisClient                   *redis.Client
	WSController                  *wsV1.Controller
	SSEHandler                    http.Handler
	Validator                     validator.Validator
	Logger                        logkit.Logger
	TrustedProxyCIDRs             []string
	StructuredLogger              bool
	DebugEnabled                  bool
	RateLimitConfigCache          *middleware.RateLimitConfigCache
	ForgotPasswordRateLimiter     *middleware.PerKeyRateLimiter
	ResendVerificationRateLimiter *middleware.PerKeyRateLimiter
	ResetPasswordTokenRateLimiter *middleware.PerKeyRateLimiter
	RatelimitAuditWG              *sync.WaitGroup
	TM                            repo.TransactionManager
}

type ServerDeps struct {
	Challenge ChallengeDeps
	Team      TeamDeps
	User      UserDeps
	Comp      CompetitionDeps
	Admin     AdminDeps
	Infra     InfraDeps
}
