package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

func NewRouter(
	router chi.Router,
	deps *helper.ServerDeps,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
	competitionMode string,
) {
	server := NewServer(deps)
	wrapper := openapi.ServerInterfaceWrapper{
		Handler: server,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			helper.RenderError(w, r, http.StatusBadRequest, err.Error())
		},
	}
	setupPublicRoutes(router, server, wrapper, deps.Infra.RedisClient, deps.Infra.Logger, deps.Infra.TrustedProxyCIDRs)
	setupAuthOnlyRoutes(router, deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, wrapper)
	setupProtectedRoutes(router, server, deps, wrapper, submitLimit, durationLimit, verifyEmails, competitionMode)
}

func setupPublicRoutes(router chi.Router, server *Server, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, logger logger.Logger, trustedProxyCIDRs []string) {
	loginLimit := restapimiddleware.RateLimit(redisClient, "auth:login:ip", 10, time.Minute, func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, trustedProxyCIDRs), nil
	}, logger)
	registerLimit := restapimiddleware.RateLimit(redisClient, "auth:register:ip", 5, time.Minute, func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, trustedProxyCIDRs), nil
	}, logger)
	forgotPasswordLimit := restapimiddleware.RateLimit(redisClient, "auth:forgot:ip", 3, time.Minute, func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, trustedProxyCIDRs), nil
	}, logger)
	resetPasswordLimit := restapimiddleware.RateLimit(redisClient, "auth:reset:ip", 5, time.Minute, func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, trustedProxyCIDRs), nil
	}, logger)
	logoutLimit := restapimiddleware.RateLimit(redisClient, "auth:logout:ip", 10, time.Minute, func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, trustedProxyCIDRs), nil
	}, logger)

	router.Group(func(r chi.Router) {
		// Auth endpoints with rate limiting
		r.With(loginLimit).Post("/auth/login", wrapper.PostAuthLogin)
		r.With(registerLimit).Post("/auth/register", wrapper.PostAuthRegister)
		r.Get("/auth/verify-email", wrapper.GetAuthVerifyEmail)
		r.With(forgotPasswordLimit).Post("/auth/forgot-password", wrapper.PostAuthForgotPassword)
		r.With(resetPasswordLimit).Post("/auth/reset-password", wrapper.PostAuthResetPassword)
		r.With(logoutLimit).Post("/auth/logout", server.PostAuthLogout)

		// Public endpoints
		r.Get("/competition/status", wrapper.GetCompetitionStatus)
		r.Get("/users/{ID}", wrapper.GetUsersID)
		r.Get("/tags", wrapper.GetTags)
		r.Get("/fields", wrapper.GetFields)
		r.Get("/brackets", wrapper.GetBrackets)
		r.Get("/ratings", wrapper.GetRatings)
		r.Get("/ratings/team/{ID}", wrapper.GetRatingsTeamID)
		r.Get("/pages", wrapper.GetPages)
		r.Get("/pages/{slug}", wrapper.GetPagesSlug)
		r.Get("/notifications", wrapper.GetNotifications)
	})
}

func setupAuthOnlyRoutes(router chi.Router, jwtService *jwt.JWTService, apiTokenUC usecase.APITokenUseCase, userUC *user.UserUseCase, wrapper openapi.ServerInterfaceWrapper) {
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(jwtService, apiTokenUC, userUC))

		r.Post("/auth/resend-verification", wrapper.PostAuthResendVerification)
	})
}

func setupProtectedRoutes(
	router chi.Router,
	server *Server,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
	competitionMode string,
) {
	// Basic authenticated routes
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC))

		r.Get("/auth/me", wrapper.GetAuthMe)

		r.Get("/user/notifications", wrapper.GetUserNotifications)
		r.Patch("/user/notifications/{ID}/read", wrapper.PatchUserNotificationsIDRead)
		r.Get("/user/tokens", wrapper.GetUserTokens)
		r.Post("/user/tokens", wrapper.PostUserTokens)
		r.Delete("/user/tokens/{ID}", wrapper.DeleteUserTokensID)

		setupTeamRoutes(r, wrapper, verifyEmails, competitionMode)
		setupChallengeRoutes(r, wrapper, deps.Comp.CompetitionUC, deps.Challenge.CommentUC, deps.Infra.RedisClient, deps.Infra.TrustedProxyCIDRs, submitLimit, durationLimit, verifyEmails, competitionMode, deps.Infra.Logger)

		r.Get("/files/{ID}/download", wrapper.GetFilesIDDownload)

		setupAdminRoutes(r, wrapper)
	})

	// Scoreboard and Statistics (require Auth + ScoreboardVisibility)
	scoreboardLimit := restapimiddleware.RateLimit(deps.Infra.RedisClient, "scoreboard:ip", 30, time.Minute, func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, deps.Infra.TrustedProxyCIDRs), nil
	}, deps.Infra.Logger)

	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC))
		r.Use(restapimiddleware.ScoreboardVisibility(deps.Admin.AppSettingsRepo))

		r.With(scoreboardLimit).Get("/scoreboard", wrapper.GetScoreboard)
		r.Get("/scoreboard/graph", wrapper.GetScoreboardGraph)
		r.Get("/statistics/general", wrapper.GetStatisticsGeneral)
		r.Get("/statistics/challenges", wrapper.GetStatisticsChallenges)
		r.Get("/statistics/challenges/{id}", wrapper.GetStatisticsChallengesId)
		r.Get("/statistics/scoreboard", wrapper.GetStatisticsScoreboard)
	})

	// First Blood endpoint (require Auth + ChallengeVisibility)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC))
		r.Use(restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC))

		r.Get("/challenges/{ID}/first-blood", wrapper.GetChallengesIDFirstBlood)
	})

	// WebSocket (require Auth)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC))

		r.Get("/ws", wrapper.GetWs)
	})

	// Direct File Download (require Auth + RequireVerified + path validation)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC))
		r.Use(restapimiddleware.RequireVerified(verifyEmails))
		r.Use(restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC))

		r.Get("/files/download/*", server.Download)
	})
}

func setupTeamRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, verifyEmails bool, _ string) {
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
	_ *challenge.CommentUseCase,
	redisClient *redis.Client,
	trustedProxyCIDRs []string,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
	competitionMode string,
	log logger.Logger,
) {
	// Challenge endpoints with visibility check
	r.Group(func(challenges chi.Router) {
		challenges.Use(restapimiddleware.ChallengeVisibility(competitionUC))
		challenges.Get("/challenges", wrapper.GetChallenges)
		challenges.Get("/challenges/{challengeID}/files", wrapper.GetChallengesChallengeIDFiles)
		challenges.Get("/challenges/{challengeID}/hints", wrapper.GetChallengesChallengeIDHints)
	})

	r.Group(func(comments chi.Router) {
		comments.Use(restapimiddleware.CompetitionEnded(competitionUC))
		comments.Get("/challenges/{challengeID}/comments", wrapper.GetChallengesChallengeIDComments)
		comments.Post("/challenges/{challengeID}/comments", wrapper.PostChallengesChallengeIDComments)
		comments.Delete("/comments/{ID}", wrapper.DeleteCommentsID)
	})

	// Submit Flag (Rate Limited + Verification + Team)
	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(restapimiddleware.RequireVerified(verifyEmails))
		sub.Use(restapimiddleware.RequireTeam(competitionMode))

		ipLimit := restapimiddleware.RateLimit(redisClient, "submit:ip", int64(submitLimit*3), durationLimit, func(r *http.Request) (string, error) {
			return helper.GetClientIP(r, trustedProxyCIDRs), nil
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
	sub := r.With(restapimiddleware.RequireVerified(verifyEmails), restapimiddleware.RequireTeam(competitionMode))
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
		adm.Get("/admin/configs", wrapper.GetAdminConfigs)
		adm.Get("/admin/configs/{key}", wrapper.GetAdminConfigsKey)
		adm.Put("/admin/configs/{key}", wrapper.PutAdminConfigsKey)
		adm.Delete("/admin/configs/{key}", wrapper.DeleteAdminConfigsKey)

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
		adm.Patch("/admin/teams/{ID}/bracket", wrapper.PatchAdminTeamsIDBracket)

		// Admin Brackets
		adm.Post("/admin/brackets", wrapper.PostAdminBrackets)
		adm.Get("/admin/brackets/{ID}", wrapper.GetAdminBracketsID)
		adm.Put("/admin/brackets/{ID}", wrapper.PutAdminBracketsID)
		adm.Delete("/admin/brackets/{ID}", wrapper.DeleteAdminBracketsID)

		// Admin CTF Events / Ratings
		adm.Get("/admin/ctf-events", wrapper.GetAdminCtfEvents)
		adm.Post("/admin/ctf-events", wrapper.PostAdminCtfEvents)
		adm.Post("/admin/ctf-events/{ID}/finalize", wrapper.PostAdminCtfEventsIDFinalize)

		// Admin Tags
		adm.Post("/admin/tags", wrapper.PostAdminTags)
		adm.Put("/admin/tags/{ID}", wrapper.PutAdminTagsID)
		adm.Delete("/admin/tags/{ID}", wrapper.DeleteAdminTagsID)

		// Admin Fields
		adm.Post("/admin/fields", wrapper.PostAdminFields)
		adm.Put("/admin/fields/{ID}", wrapper.PutAdminFieldsID)
		adm.Delete("/admin/fields/{ID}", wrapper.DeleteAdminFieldsID)

		// Admin Pages
		adm.Get("/admin/pages", wrapper.GetAdminPages)
		adm.Post("/admin/pages", wrapper.PostAdminPages)
		adm.Get("/admin/pages/{ID}", wrapper.GetAdminPagesID)
		adm.Put("/admin/pages/{ID}", wrapper.PutAdminPagesID)
		adm.Delete("/admin/pages/{ID}", wrapper.DeleteAdminPagesID)

		// Admin Notifications
		adm.Post("/admin/notifications", wrapper.PostAdminNotifications)
		adm.Post("/admin/notifications/user/{userID}", wrapper.PostAdminNotificationsUserUserID)
		adm.Put("/admin/notifications/{ID}", wrapper.PutAdminNotificationsID)
		adm.Delete("/admin/notifications/{ID}", wrapper.DeleteAdminNotificationsID)

		// Admin Submissions
		adm.Get("/admin/submissions", wrapper.GetAdminSubmissions)
		adm.Get("/admin/submissions/challenge/{challengeID}", wrapper.GetAdminSubmissionsChallengeChallengeID)
		adm.Get("/admin/submissions/challenge/{challengeID}/stats", wrapper.GetAdminSubmissionsChallengeChallengeIDStats)
		adm.Get("/admin/submissions/user/{userID}", wrapper.GetAdminSubmissionsUserUserID)
		adm.Get("/admin/submissions/team/{teamID}", wrapper.GetAdminSubmissionsTeamTeamID)

		// Admin Backup
		adm.Get("/admin/export", wrapper.GetAdminExport)
		adm.Get("/admin/export/zip", wrapper.GetAdminExportZip)
		adm.Post("/admin/import", wrapper.PostAdminImport)
	})
}
