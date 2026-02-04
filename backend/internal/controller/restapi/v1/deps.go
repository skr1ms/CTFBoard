package v1

import (
	"github.com/redis/go-redis/v9"
	wsV1 "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/internal/usecase/email"
	"github.com/skr1ms/CTFBoard/internal/usecase/page"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	"github.com/skr1ms/CTFBoard/internal/usecase/team"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type ServerDeps struct {
	UserUC          *user.UserUseCase
	ChallengeUC     *challenge.ChallengeUseCase
	SolveUC         *competition.SolveUseCase
	TeamUC          *team.TeamUseCase
	CompetitionUC   *competition.CompetitionUseCase
	HintUC          *challenge.HintUseCase
	EmailUC         *email.EmailUseCase
	FileUC          *challenge.FileUseCase
	AwardUC         *team.AwardUseCase
	StatsUC         *competition.StatisticsUseCase
	SubmissionUC    *competition.SubmissionUseCase
	TagUC           *challenge.TagUseCase
	FieldUC         *settings.FieldUseCase
	PageUC          *page.PageUseCase
	BracketUC      *competition.BracketUseCase
	NotifUC         usecase.NotificationUseCase
	APITokenUC      usecase.APITokenUseCase
	BackupUC        usecase.BackupUseCase
	SettingsUC      *settings.SettingsUseCase
	DynamicConfigUC *competition.DynamicConfigUseCase
	CommentUC       *challenge.CommentUseCase
	RatingUC        *competition.RatingUseCase
	JWTService      *jwt.JWTService
	RedisClient     *redis.Client
	WSController    *wsV1.Controller
	Validator       validator.Validator
	Logger          logger.Logger
}
