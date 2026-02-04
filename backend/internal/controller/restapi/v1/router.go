package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
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
	deps *ServerDeps,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
) {
	server := NewServer(deps)
	wrapper := openapi.ServerInterfaceWrapper{
		Handler: server,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			RenderError(w, r, http.StatusBadRequest, err.Error())
		},
	}
	setupPublicRoutes(router, server, wrapper, deps.RedisClient, deps.Logger)
	setupAuthOnlyRoutes(router, deps.JWTService, deps.APITokenUC, deps.UserUC, wrapper)
	setupProtectedRoutes(router, deps, wrapper, submitLimit, durationLimit, verifyEmails)
}

func setupPublicRoutes(router chi.Router, server *Server, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, logger logger.Logger) {
	scoreboardLimit := restapimiddleware.RateLimit(redisClient, "scoreboard:ip", 30, time.Minute, func(r *http.Request) (string, error) {
		return GetClientIP(r), nil
	}, logger)

	router.Group(func(r chi.Router) {
		r.Post("/auth/login", wrapper.PostAuthLogin)
		r.Post("/auth/register", wrapper.PostAuthRegister)
		r.Get("/auth/verify-email", wrapper.GetAuthVerifyEmail)
		r.Post("/auth/forgot-password", wrapper.PostAuthForgotPassword)
		r.Post("/auth/reset-password", wrapper.PostAuthResetPassword)

		r.Get("/competition/status", wrapper.GetCompetitionStatus)
		r.With(scoreboardLimit).Get("/scoreboard", wrapper.GetScoreboard)
		r.Get("/challenges/{ID}/first-blood", wrapper.GetChallengesIDFirstBlood)
		r.Get("/users/{ID}", wrapper.GetUsersID)
		r.Get("/tags", wrapper.GetTags)
		r.Get("/fields", wrapper.GetFields)
		r.Get("/brackets", wrapper.GetBrackets)
		r.Get("/ratings", wrapper.GetRatings)
		r.Get("/ratings/team/{ID}", wrapper.GetRatingsTeamID)
		r.Get("/pages", wrapper.GetPages)
		r.Get("/pages/{slug}", wrapper.GetPagesSlug)
		r.Get("/notifications", wrapper.GetNotifications)
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

func setupAuthOnlyRoutes(router chi.Router, jwtService *jwt.JWTService, apiTokenUC usecase.APITokenUseCase, userUC *user.UserUseCase, wrapper openapi.ServerInterfaceWrapper) {
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(jwtService, apiTokenUC, userUC))

		r.Post("/auth/resend-verification", wrapper.PostAuthResendVerification)
	})
}

func setupProtectedRoutes(
	router chi.Router,
	deps *ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
) {
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.JWTService, deps.APITokenUC, deps.UserUC))
		r.Use(restapimiddleware.InjectUser(deps.UserUC))

		r.Get("/auth/me", wrapper.GetAuthMe)

		r.Get("/user/notifications", wrapper.GetUserNotifications)
		r.Patch("/user/notifications/{ID}/read", wrapper.PatchUserNotificationsIDRead)
		r.Get("/user/tokens", wrapper.GetUserTokens)
		r.Post("/user/tokens", wrapper.PostUserTokens)
		r.Delete("/user/tokens/{ID}", wrapper.DeleteUserTokensID)

		setupTeamRoutes(r, wrapper, verifyEmails)
		setupChallengeRoutes(r, wrapper, deps.CompetitionUC, deps.CommentUC, deps.RedisClient, submitLimit, durationLimit, verifyEmails, deps.Logger)

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
	_ *challenge.CommentUseCase,
	redisClient *redis.Client,
	submitLimit int,
	durationLimit time.Duration,
	verifyEmails bool,
	log logger.Logger,
) {
	r.Get("/challenges", wrapper.GetChallenges)
	r.Get("/challenges/{challengeID}/files", wrapper.GetChallengesChallengeIDFiles)
	r.Get("/challenges/{challengeID}/hints", wrapper.GetChallengesChallengeIDHints)

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
