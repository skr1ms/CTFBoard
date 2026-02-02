package v1

import (
	"github.com/redis/go-redis/v9"
	wsV1 "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/internal/usecase/email"
	"github.com/skr1ms/CTFBoard/internal/usecase/team"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type Server struct {
	openapi.Unimplemented
	userUC        *user.UserUseCase
	challengeUC   *challenge.ChallengeUseCase
	solveUC       *competition.SolveUseCase
	teamUC        *team.TeamUseCase
	competitionUC *competition.CompetitionUseCase
	hintUC        *challenge.HintUseCase
	emailUC       *email.EmailUseCase
	fileUC        *challenge.FileUseCase
	awardUC       *team.AwardUseCase
	statsUC       *competition.StatisticsUseCase
	backupUC      usecase.BackupUseCase
	settingsUC    *competition.SettingsUseCase
	jwtService    *jwt.JWTService
	redisClient   *redis.Client
	wsController  *wsV1.Controller
	validator     validator.Validator
	logger        logger.Logger
}

func NewServer(
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
	backupUC usecase.BackupUseCase,
	settingsUC *competition.SettingsUseCase,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
	wsController *wsV1.Controller,
	validator validator.Validator,
	logger logger.Logger,
) *Server {
	return &Server{
		userUC:        userUC,
		challengeUC:   challengeUC,
		solveUC:       solveUC,
		teamUC:        teamUC,
		competitionUC: competitionUC,
		hintUC:        hintUC,
		emailUC:       emailUC,
		fileUC:        fileUC,
		awardUC:       awardUC,
		statsUC:       statsUC,
		backupUC:      backupUC,
		settingsUC:    settingsUC,
		jwtService:    jwtService,
		redisClient:   redisClient,
		wsController:  wsController,
		validator:     validator,
		logger:        logger,
	}
}
