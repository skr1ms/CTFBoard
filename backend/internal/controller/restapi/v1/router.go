package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	ws "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
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

func NewRouter(
	router chi.Router,
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
	wsController *ws.Controller,
	validator validator.Validator,
	logger logger.Logger,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
) {
	// Initialize Server
	server := NewServer(
		userUC, challengeUC, solveUC, teamUC, competitionUC, hintUC, emailUC, fileUC, awardUC, statsUC, backupUC,
		settingsUC, jwtService, redisClient, wsController, validator, logger,
	)

	// Wrap with OpenAPI handler
	wrapper := openapi.ServerInterfaceWrapper{
		Handler: server,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			RenderError(w, r, http.StatusBadRequest, err.Error())
		},
	}

	setupPublicRoutes(router, server, wrapper)
	setupAuthOnlyRoutes(router, jwtService, wrapper)
	setupProtectedRoutes(router, userUC, jwtService, competitionUC, redisClient, wrapper, submitLimit, durationLimit, verifyEmails, logger)
}

func setupPublicRoutes(router chi.Router, server *Server, wrapper openapi.ServerInterfaceWrapper) {
	// Public Routes
	router.Group(func(r chi.Router) {
		r.Post("/auth/login", wrapper.PostAuthLogin)
		r.Post("/auth/register", wrapper.PostAuthRegister)
		r.Get("/auth/verify-email", wrapper.GetAuthVerifyEmail)
		r.Post("/auth/forgot-password", wrapper.PostAuthForgotPassword)
		r.Post("/auth/reset-password", wrapper.PostAuthResetPassword)

		r.Get("/competition/status", wrapper.GetCompetitionStatus)
		r.Get("/scoreboard", wrapper.GetScoreboard)
		r.Get("/challenges/{ID}/first-blood", wrapper.GetChallengesIDFirstBlood)
		r.Get("/users/{ID}", wrapper.GetUsersID)
		r.Get("/statistics/general", wrapper.GetStatisticsGeneral)
		r.Get("/statistics/challenges", wrapper.GetStatisticsChallenges)
		r.Get("/statistics/challenges/{id}", wrapper.GetStatisticsChallengesId)
		r.Get("/statistics/scoreboard", wrapper.GetStatisticsScoreboard)
		r.Get("/scoreboard/graph", wrapper.GetScoreboardGraph)

		// WebSocket
		r.Get("/ws", wrapper.GetWs)

		// Direct File Download (Manual)
		r.Get("/files/download/*", server.Download)
	})
}

func setupAuthOnlyRoutes(router chi.Router, jwtService *jwt.JWTService, wrapper openapi.ServerInterfaceWrapper) {
	// Protected Routes (Auth Only)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(jwtService))

		r.Post("/auth/resend-verification", wrapper.PostAuthResendVerification)
	})
}

func setupProtectedRoutes(
	router chi.Router,
	userUC *user.UserUseCase,
	jwtService *jwt.JWTService,
	competitionUC *competition.CompetitionUseCase,
	redisClient *redis.Client,
	wrapper openapi.ServerInterfaceWrapper,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
	logger logger.Logger,
) {
	// Protected Routes (Auth + InjectUser)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(jwtService))
		r.Use(restapimiddleware.InjectUser(userUC))

		r.Get("/auth/me", wrapper.GetAuthMe)

		setupTeamRoutes(r, wrapper, verifyEmails)
		setupChallengeRoutes(r, wrapper, competitionUC, redisClient, submitLimit, durationLimit, verifyEmails, logger)

		// Files Download URL (Protected)
		r.Get("/files/{ID}/download", wrapper.GetFilesIDDownload)

		setupAdminRoutes(r, wrapper)
	})
}

func setupTeamRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, verifyEmails bool) {
	// Team
	r.Get("/teams/my", wrapper.GetTeamsMy)
	r.Get("/teams/{ID}", wrapper.GetTeamsID)
	r.Post("/teams/leave", wrapper.PostTeamsLeave)
	r.Delete("/teams/me", wrapper.DeleteTeamsMe)
	r.Delete("/teams/members/{ID}", wrapper.DeleteTeamsMembersID)
	r.Post("/teams/transfer-captain", wrapper.PostTeamsTransferCaptain)

	verified := r.With(restapimiddleware.RequireVerified(verifyEmails))
	verified.Post("/teams", wrapper.PostTeams)
	verified.Post("/teams/join", wrapper.PostTeamsJoin)
	verified.Post("/teams/solo", wrapper.PostTeamsSolo)
}

func setupChallengeRoutes(
	r chi.Router,
	wrapper openapi.ServerInterfaceWrapper,
	competitionUC *competition.CompetitionUseCase,
	redisClient *redis.Client,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
	log logger.Logger,
) {
	// Challenges
	r.Get("/challenges", wrapper.GetChallenges)
	r.Get("/challenges/{challengeID}/files", wrapper.GetChallengesChallengeIDFiles)
	r.Get("/challenges/{challengeID}/hints", wrapper.GetChallengesChallengeIDHints)

	// Submit Flag (Rate Limited + Verification + Team)
	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(restapimiddleware.RequireVerified(verifyEmails))
		sub.Use(restapimiddleware.RequireTeam(""))

		ipLimit := restapimiddleware.RateLimit(redisClient, "submit:ip", int64(submitLimit*3), durationLimit, func(r *http.Request) (string, error) {
			return GetClientIP(r), nil
		}, log)
		userLimit := restapimiddleware.RateLimit(redisClient, "submit:user", int64(submitLimit), durationLimit, func(r *http.Request) (string, error) {
			user, ok := restapimiddleware.GetUser(r.Context())
			if !ok {
				return "", http.ErrNoCookie
			}
			return user.ID.String(), nil
		}, log)

		sub.With(ipLimit, userLimit).Post("/challenges/{ID}/submit", wrapper.PostChallengesIDSubmit)
	})

	// Unlock Hints
	sub := r.With(restapimiddleware.RequireVerified(verifyEmails), restapimiddleware.RequireTeam(""))
	sub.Post("/challenges/{challengeID}/hints/{hintID}/unlock", wrapper.PostChallengesChallengeIDHintsHintIDUnlock)
}

func setupAdminRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	// Admin Routes
	r.Group(func(adm chi.Router) {
		adm.Use(restapimiddleware.Admin)

		adm.Get("/admin/competition", wrapper.GetAdminCompetition)
		adm.Put("/admin/competition", wrapper.PutAdminCompetition)
		adm.Get("/admin/settings", wrapper.GetAdminSettings)
		adm.Put("/admin/settings", wrapper.PutAdminSettings)

		// Admin Challenges
		adm.Post("/admin/challenges", wrapper.PostAdminChallenges)
		adm.Put("/admin/challenges/{ID}", wrapper.PutAdminChallengesID)
		adm.Delete("/admin/challenges/{ID}", wrapper.DeleteAdminChallengesID)
		adm.Post("/admin/challenges/{challengeID}/files", wrapper.PostAdminChallengesChallengeIDFiles)
		adm.Post("/admin/challenges/{challengeID}/hints", wrapper.PostAdminChallengesChallengeIDHints)

		// Admin Hints
		adm.Put("/admin/hints/{ID}", wrapper.PutAdminHintsID)
		adm.Delete("/admin/hints/{ID}", wrapper.DeleteAdminHintsID)

		// Admin Files
		adm.Delete("/admin/files/{ID}", wrapper.DeleteAdminFilesID)

		// Admin Awards
		adm.Post("/admin/awards", wrapper.PostAdminAwards)
		adm.Get("/admin/awards/team/{teamID}", wrapper.GetAdminAwardsTeamTeamID)

		// Admin Teams
		adm.Post("/admin/teams/{ID}/ban", wrapper.PostAdminTeamsIDBan)
		adm.Delete("/admin/teams/{ID}/ban", wrapper.DeleteAdminTeamsIDBan)
		adm.Patch("/admin/teams/{ID}/hidden", wrapper.PatchAdminTeamsIDHidden)

		// Admin Backup
		adm.Get("/admin/export", wrapper.GetAdminExport)
		adm.Get("/admin/export/zip", wrapper.GetAdminExportZip)
		adm.Post("/admin/import", wrapper.PostAdminImport)
	})
}
