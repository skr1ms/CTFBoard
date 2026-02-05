package v1

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	wsV1 "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/openapi"
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

// OnError logs err, gives the client a response via HandleError, and returns true.
// The method is declared here, not in helper/error, so as not to create an import loop v1 ↔ helper.
func (h *Server) OnError(w http.ResponseWriter, r *http.Request, err error, op, step string) bool {
	if err == nil {
		return false
	}
	h.logger.WithError(err).Error("restapi - v1 - " + op + " - " + step)
	helper.HandleError(w, r, err)
	return true
}

type Server struct {
	openapi.Unimplemented
	userUC          *user.UserUseCase
	challengeUC     *challenge.ChallengeUseCase
	solveUC         *competition.SolveUseCase
	teamUC          *team.TeamUseCase
	competitionUC   *competition.CompetitionUseCase
	hintUC          *challenge.HintUseCase
	emailUC         *email.EmailUseCase
	fileUC          *challenge.FileUseCase
	awardUC         *team.AwardUseCase
	statsUC         *competition.StatisticsUseCase
	submissionUC    *competition.SubmissionUseCase
	tagUC           *challenge.TagUseCase
	fieldUC         *settings.FieldUseCase
	pageUC          *page.PageUseCase
	bracketUC       *competition.BracketUseCase
	notifUC         usecase.NotificationUseCase
	apiTokenUC      usecase.APITokenUseCase
	backupUC        usecase.BackupUseCase
	settingsUC      *settings.SettingsUseCase
	dynamicConfigUC *competition.DynamicConfigUseCase
	commentUC       *challenge.CommentUseCase
	ratingUC        *competition.RatingUseCase
	jwtService      *jwt.JWTService
	redisClient     *redis.Client
	wsController    *wsV1.Controller
	validator       validator.Validator
	logger          logger.Logger
}

func NewServer(deps *helper.ServerDeps) *Server {
	if deps == nil {
		return nil
	}
	return &Server{
		userUC:          deps.UserUC,
		challengeUC:     deps.ChallengeUC,
		solveUC:         deps.SolveUC,
		teamUC:          deps.TeamUC,
		competitionUC:   deps.CompetitionUC,
		hintUC:          deps.HintUC,
		emailUC:         deps.EmailUC,
		fileUC:          deps.FileUC,
		awardUC:         deps.AwardUC,
		statsUC:         deps.StatsUC,
		submissionUC:    deps.SubmissionUC,
		tagUC:           deps.TagUC,
		fieldUC:         deps.FieldUC,
		pageUC:          deps.PageUC,
		bracketUC:       deps.BracketUC,
		notifUC:         deps.NotifUC,
		apiTokenUC:      deps.APITokenUC,
		backupUC:        deps.BackupUC,
		settingsUC:      deps.SettingsUC,
		dynamicConfigUC: deps.DynamicConfigUC,
		commentUC:       deps.CommentUC,
		ratingUC:        deps.RatingUC,
		jwtService:      deps.JWTService,
		redisClient:     deps.RedisClient,
		wsController:    deps.WSController,
		validator:       deps.Validator,
		logger:          deps.Logger,
	}
}
