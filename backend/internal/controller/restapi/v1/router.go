package v1

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
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
	awardUC *usecase.AwardUseCase,
	statsUC *usecase.StatisticsUseCase,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
	validator validator.Validator,
	logger logger.Logger,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
) {
	authRouter := chi.NewRouter()
	router.Mount("/auth", authRouter)

	// Public / Auth routes
	NewUserRoutes(router, authRouter, userUC, emailUC, validator, logger, jwtService)
	NewEmailRoutes(authRouter, emailUC, validator, logger, jwtService, redisClient)
	NewScoreboardRoutes(router, solveUC, logger)

	// Protected routes (Auth + InjectUser)
	router.Group(func(r chi.Router) {
		r.Use(restapiMiddleware.Auth(jwtService))
		r.Use(restapiMiddleware.InjectUser(userUC))

		NewChallengeRoutes(r, challengeUC, solveUC, userUC, competitionUC, validator, logger, jwtService, redisClient, submitLimit, durationLimit, verifyEmails)
		NewTeamRoutes(r, teamUC, userUC, validator, logger, jwtService, redisClient)
		NewCompetitionRoutes(router, r, competitionUC, userUC, validator, logger, jwtService)
		NewHintRoutes(r, hintUC, userUC, validator, logger, jwtService, verifyEmails)
		NewFileRoutes(r, fileUC, logger, jwtService)
		NewAwardRoutes(r, awardUC, validator, logger, jwtService)
		NewStatisticsRoutes(r, statsUC, logger, jwtService)
	})
}
