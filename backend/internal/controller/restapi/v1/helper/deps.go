package helper

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

type ChallengeDeps struct {
	ChallengeUC *challenge.ChallengeUseCase
	HintUC      *challenge.HintUseCase
	FileUC      *challenge.FileUseCase
	TagUC       *challenge.TagUseCase
	CommentUC   *challenge.CommentUseCase
}

type TeamDeps struct {
	TeamUC  *team.TeamUseCase
	AwardUC *team.AwardUseCase
}

type UserDeps struct {
	UserUC     *user.UserUseCase
	EmailUC    *email.EmailUseCase
	APITokenUC usecase.APITokenUseCase
}

type CompetitionDeps struct {
	CompetitionUC *competition.CompetitionUseCase
	SolveUC       *competition.SolveUseCase
	StatsUC       *competition.StatisticsUseCase
	SubmissionUC  *competition.SubmissionUseCase
	BracketUC     *competition.BracketUseCase
	RatingUC      *competition.RatingUseCase
}

type AdminDeps struct {
	BackupUC        usecase.BackupUseCase
	SettingsUC      *settings.SettingsUseCase
	DynamicConfigUC *competition.DynamicConfigUseCase
	FieldUC         *settings.FieldUseCase
	PageUC          *page.PageUseCase
	NotifUC         usecase.NotificationUseCase
}

type InfraDeps struct {
	JWTService   *jwt.JWTService
	RedisClient  *redis.Client
	WSController *wsV1.Controller
	Validator    validator.Validator
	Logger       logger.Logger
}

type ServerDeps struct {
	Challenge ChallengeDeps
	Team      TeamDeps
	User      UserDeps
	Comp      CompetitionDeps
	Admin     AdminDeps
	Infra     InfraDeps
}
