package v1

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-logkit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// setupAdminRoutes applies the Admin role-check middleware and a shared per-user
// general rate limiter (adminGeneralLimit / adminGeneralWindow) to the admin
// subtree, then delegates to 11 sub-group helpers covering: configs, challenges,
// awards, users, teams, brackets, tags, fields, pages, notifications, and utility
// (export, import, reset, debug) - the last group adds its own tighter destructive
// and export-zip rate limits.
func setupAdminRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, log logkit.Logger) {
	adminGeneralLimitMw := restapimiddleware.RateLimit(redisClient, rlKeyAdminGeneral, adminGeneralLimit, adminGeneralWindow, userIDKeyFunc, log)

	r.Group(func(adm chi.Router) {
		adm.Use(restapimiddleware.Admin)
		adm.Use(adminGeneralLimitMw)

		setupAdminConfigRoutes(adm, wrapper)
		setupAdminChallengeRoutes(adm, wrapper)
		setupAdminAwardRoutes(adm, wrapper)
		setupAdminUserRoutes(adm, wrapper)
		setupAdminTeamRoutes(adm, wrapper)
		setupAdminBracketRoutes(adm, wrapper)
		setupAdminTagRoutes(adm, wrapper)
		setupAdminFieldRoutes(adm, wrapper)
		setupAdminPageRoutes(adm, wrapper)
		setupAdminNotificationRoutes(adm, wrapper)
		setupAdminSubmissionRoutes(adm, wrapper)
		setupAdminAppealRoutes(adm, wrapper)
		setupAdminStorageRoutes(adm, wrapper)
		setupAdminUtilityRoutes(adm, wrapper, redisClient, log)
	})
}

func setupAdminAppealRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/appeals", wrapper.GetAdminAppeals)
	adm.Patch("/admin/appeals/{ID}", wrapper.PatchAdminAppealsID)
}

func setupAdminStorageRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/storage", wrapper.GetAdminStorage)
	adm.Delete("/admin/storage/{path}", wrapper.DeleteAdminStoragePath)
}

func setupAdminConfigRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/competition", wrapper.GetAdminCompetition)
	adm.Put("/admin/competition", wrapper.PutAdminCompetition)
	adm.Get("/admin/settings", wrapper.GetAdminSettings)
	adm.Put("/admin/settings", wrapper.PutAdminSettings)
	adm.Get("/admin/configs", wrapper.GetAdminConfigs)
	adm.Get("/admin/configs/categories", wrapper.GetAdminConfigsCategories)
	adm.Get("/admin/configs/category/{category}", wrapper.GetAdminConfigsCategory)
	adm.Put("/admin/configs/batch", wrapper.PutAdminConfigsBatch)
	adm.Get("/admin/configs/{key}", wrapper.GetAdminConfigsKey)
	adm.Put("/admin/configs/{key}", wrapper.PutAdminConfigsKey)
	adm.Delete("/admin/configs/{key}", wrapper.DeleteAdminConfigsKey)
}

func setupAdminChallengeRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/challenges", wrapper.PostAdminChallenges)
	adm.Put("/admin/challenges/{ID}", wrapper.PutAdminChallengesID)
	adm.Delete("/admin/challenges/{ID}", wrapper.DeleteAdminChallengesID)
	adm.Post("/admin/challenges/{challengeID}/files", wrapper.PostAdminChallengesChallengeIDFiles)
	adm.Post("/admin/challenges/{challengeID}/hints", wrapper.PostAdminChallengesChallengeIDHints)
	adm.Get("/admin/challenges/{challengeID}/flags", wrapper.GetAdminChallengesChallengeIDFlags)
	adm.Put("/admin/challenges/{challengeID}/requirements", wrapper.PutAdminChallengesChallengeIDRequirements)
	adm.Post("/admin/challenges/{challengeID}/solution", wrapper.PostAdminChallengesChallengeIDSolution)
	adm.Delete("/admin/challenges/{challengeID}/solution", wrapper.DeleteAdminChallengesChallengeIDSolution)
	adm.Post("/admin/challenges/recalc-points", wrapper.PostAdminChallengesRecalcPoints)
}

func setupAdminAwardRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/awards", wrapper.GetAdminAwards)
	adm.Post("/admin/awards", wrapper.PostAdminAwards)
	adm.Get("/admin/awards/{ID}", wrapper.GetAdminAwardsID)
	adm.Delete("/admin/awards/{ID}", wrapper.DeleteAdminAwardsID)
	adm.Get("/admin/awards/team/{teamID}", wrapper.GetAdminAwardsTeamTeamID)
}

func setupAdminUserRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/users", wrapper.GetAdminUsers)
	adm.Post("/admin/users", wrapper.PostAdminUsers)
	adm.Patch("/admin/users/{ID}", wrapper.PatchAdminUsersID)
	adm.Delete("/admin/users/{ID}", wrapper.DeleteAdminUsersID)
	adm.Get("/admin/users/{ID}/tracking", wrapper.GetAdminUsersIDTracking)
	adm.Get("/admin/users/{ID}/missing-challenges", wrapper.GetAdminUsersIDMissingChallenges)
	adm.Post("/admin/users/{ID}/ban", wrapper.PostAdminUsersIDBan)
	adm.Delete("/admin/users/{ID}/ban", wrapper.DeleteAdminUsersIDBan)
	adm.Put("/admin/users/{ID}/avatar", wrapper.PutAdminUsersIDAvatar)
	adm.Delete("/admin/users/{ID}/avatar", wrapper.DeleteAdminUsersIDAvatar)
}

func setupAdminTeamRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/teams", wrapper.GetAdminTeams)
	adm.Patch("/admin/teams/{ID}", wrapper.PatchAdminTeamsID)
	adm.Delete("/admin/teams/{ID}", wrapper.DeleteAdminTeamsID)
	adm.Get("/admin/teams/{ID}/members", wrapper.GetAdminTeamsIDMembers)
	adm.Post("/admin/teams/{ID}/members", wrapper.PostAdminTeamsIDMembers)
	adm.Delete("/admin/teams/{ID}/members/{userID}", wrapper.DeleteAdminTeamsIDMembersUserID)
	adm.Get("/admin/teams/{ID}/missing-challenges", wrapper.GetAdminTeamsIDMissingChallenges)
	adm.Post("/admin/teams/{ID}/ban", wrapper.PostAdminTeamsIDBan)
	adm.Delete("/admin/teams/{ID}/ban", wrapper.DeleteAdminTeamsIDBan)
	adm.Patch("/admin/teams/{ID}/hidden", wrapper.PatchAdminTeamsIDHidden)
	adm.Patch("/admin/teams/{ID}/bracket", wrapper.PatchAdminTeamsIDBracket)
	adm.Put("/admin/teams/{ID}/avatar", wrapper.PutAdminTeamsIDAvatar)
	adm.Delete("/admin/teams/{ID}/avatar", wrapper.DeleteAdminTeamsIDAvatar)
}

func setupAdminBracketRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/brackets", wrapper.PostAdminBrackets)
	adm.Get("/admin/brackets/{ID}", wrapper.GetAdminBracketsID)
	adm.Put("/admin/brackets/{ID}", wrapper.PutAdminBracketsID)
	adm.Delete("/admin/brackets/{ID}", wrapper.DeleteAdminBracketsID)
}

func setupAdminTagRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/tags", wrapper.PostAdminTags)
	adm.Put("/admin/tags/{ID}", wrapper.PutAdminTagsID)
	adm.Delete("/admin/tags/{ID}", wrapper.DeleteAdminTagsID)
}

func setupAdminFieldRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/fields", wrapper.PostAdminFields)
	adm.Put("/admin/fields/{ID}", wrapper.PutAdminFieldsID)
	adm.Delete("/admin/fields/{ID}", wrapper.DeleteAdminFieldsID)
}

func setupAdminPageRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/pages", wrapper.GetAdminPages)
	adm.Post("/admin/pages", wrapper.PostAdminPages)
	adm.Get("/admin/pages/{ID}", wrapper.GetAdminPagesID)
	adm.Put("/admin/pages/{ID}", wrapper.PutAdminPagesID)
	adm.Delete("/admin/pages/{ID}", wrapper.DeleteAdminPagesID)
}

func setupAdminNotificationRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/notifications", wrapper.PostAdminNotifications)
	adm.Post("/admin/notifications/user/{userID}", wrapper.PostAdminNotificationsUserUserID)
	adm.Put("/admin/notifications/{ID}", wrapper.PutAdminNotificationsID)
	adm.Delete("/admin/notifications/{ID}", wrapper.DeleteAdminNotificationsID)
}

func setupAdminSubmissionRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/submissions", wrapper.GetAdminSubmissions)
	adm.Post("/admin/submissions", wrapper.PostAdminSubmissions)
	adm.Get("/admin/submissions/{ID}", wrapper.GetAdminSubmissionsID)
	adm.Patch("/admin/submissions/{ID}", wrapper.PatchAdminSubmissionsID)
	adm.Delete("/admin/submissions/{ID}", wrapper.DeleteAdminSubmissionsID)
	adm.Get("/admin/submissions/challenge/{challengeID}", wrapper.GetAdminSubmissionsChallengeChallengeID)
	adm.Get("/admin/submissions/challenge/{challengeID}/stats", wrapper.GetAdminSubmissionsChallengeChallengeIDStats)
	adm.Get("/admin/submissions/user/{userID}", wrapper.GetAdminSubmissionsUserUserID)
	adm.Get("/admin/submissions/team/{teamID}", wrapper.GetAdminSubmissionsTeamTeamID)
}

func setupAdminUtilityRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, log logkit.Logger) {
	adm.Put("/admin/hints/{ID}", wrapper.PutAdminHintsID)
	adm.Delete("/admin/hints/{ID}", wrapper.DeleteAdminHintsID)
	adm.Delete("/admin/files/{ID}", wrapper.DeleteAdminFilesID)
	adm.Get("/admin/unlocks", wrapper.GetAdminUnlocks)
	adm.Get("/admin/statistics/solve-matrix", wrapper.GetAdminStatisticsSolveMatrix)

	destructiveLimit := restapimiddleware.RateLimit(redisClient, rlKeyAdminDestructive, adminDestructiveLimit, adminDestructiveWindow, userIDKeyFunc, log)
	adm.With(destructiveLimit).Post("/admin/reset", wrapper.PostAdminReset)
	adm.With(destructiveLimit).Post("/admin/import", wrapper.PostAdminImport)
	adm.With(destructiveLimit).Post("/admin/import/csv", wrapper.PostAdminImportCsv)
	adm.Get("/admin/export", wrapper.GetAdminExport)

	exportZipLimit := restapimiddleware.RateLimit(redisClient, rlKeyAdminExportZip, adminExportZipLimit, adminExportZipWindow, userIDKeyFunc, log)
	adm.With(exportZipLimit).Get("/admin/export/zip", wrapper.GetAdminExportZip)
	adm.Get("/admin/export/csv", wrapper.GetAdminExportCsv)
	adm.Get("/debug", wrapper.GetDebug)
}
