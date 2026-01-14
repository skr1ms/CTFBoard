package v1

import (
	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter(
	router chi.Router,
	userUC *usecase.UserUseCase,
	challengeUC *usecase.ChallengeUseCase,
	solveUC *usecase.SolveUseCase,
	teamUC *usecase.TeamUseCase,
	jwtService *jwt.JWTService,
	validator validator.Validator,
	logger logger.Interface,
) {
	router.Get("/swagger/*", httpSwagger.Handler())

	router.Route("/api/v1", func(r chi.Router) {
		NewUserRoutes(r, userUC, validator, logger, jwtService)
		NewScoreboardRoutes(r, solveUC, logger)
		NewEventsRoutes(r, solveUC, logger)
		NewChallengeRoutes(r, challengeUC, solveUC, userUC, validator, logger, jwtService)
		NewTeamRoutes(r, teamUC, validator, logger, jwtService)
	})
}
