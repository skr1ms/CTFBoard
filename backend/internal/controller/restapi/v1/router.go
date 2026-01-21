package v1

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

func NewRouter(
	router chi.Router,
	userUC *usecase.UserUseCase,
	challengeUC *usecase.ChallengeUseCase,
	solveUC *usecase.SolveUseCase,
	teamUC *usecase.TeamUseCase,
	competitionUC *usecase.CompetitionUseCase,
	hintUC *usecase.HintUseCase,
	emailUC *usecase.EmailUseCase,
	fileUC *usecase.FileUseCase,
	jwtService *jwt.JWTService,
	redisClient redis.Client,
	validator validator.Validator,
	logger logger.Interface,
	submitLimit int,
	durationLimit time.Duration,
) {
	authRouter := chi.NewRouter()
	router.Mount("/auth", authRouter)

	NewUserRoutes(router, authRouter, userUC, emailUC, validator, logger, jwtService)
	NewScoreboardRoutes(router, solveUC, logger)
	NewChallengeRoutes(router, challengeUC, solveUC, userUC, competitionUC, validator, logger, jwtService, submitLimit, durationLimit)
	NewTeamRoutes(router, teamUC, validator, logger, jwtService)
	NewCompetitionRoutes(router, competitionUC, validator, logger, jwtService)
	NewHintRoutes(router, hintUC, userUC, validator, logger, jwtService)
	NewEmailRoutes(authRouter, emailUC, validator, logger, jwtService, redisClient)
	NewFileRoutes(router, fileUC, logger, jwtService)
}
