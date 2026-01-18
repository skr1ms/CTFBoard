package v1

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	httpSwagger "github.com/swaggo/http-swagger"
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
	jwtService *jwt.JWTService,
	redisClient redis.Client,
	validator validator.Validator,
	logger logger.Interface,
	submitLimit int,
	durationLimit time.Duration,
) {
	router.Get("/swagger/*", httpSwagger.Handler())

	router.Route("/api/v1", func(r chi.Router) {
		authRouter := chi.NewRouter()
		r.Mount("/auth", authRouter)

		NewUserRoutes(r, authRouter, userUC, emailUC, validator, logger, jwtService)
		NewScoreboardRoutes(r, solveUC, logger)
		NewChallengeRoutes(r, challengeUC, solveUC, userUC, competitionUC, validator, logger, jwtService, submitLimit, durationLimit)
		NewTeamRoutes(r, teamUC, validator, logger, jwtService)
		NewCompetitionRoutes(r, competitionUC, validator, logger, jwtService)
		NewHintRoutes(r, hintUC, userUC, validator, logger, jwtService)
		NewEmailRoutes(authRouter, emailUC, validator, logger, jwtService, redisClient)
	})
}
