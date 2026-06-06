package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-cachekit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// setupChallengeRoutes wires auth-required challenge routes:
// (1) comment/rating endpoints gated by CompetitionEnded / RequireVerified;
// (2) submit and hint-unlock behind CompetitionActive + RequireTeam +
// SubmitRateLimitWithAudit (dual IP+user rate limit with async audit log on excess).
// Read-only challenge endpoints live in setupConditionalPublicRoutes.
func setupChallengeRoutes(
	r chi.Router,
	wrapper openapi.ServerInterfaceWrapper,
	deps *helper.ServerDeps,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
	requireVerified func(http.Handler) http.Handler,
	_ *cachekit.Cache,
	notBanned func(http.Handler) http.Handler,
) {
	competitionUC := deps.Comp.CompetitionUC
	log := deps.Infra.Logger
	getter := deps.Admin.SettingsUC
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, getter, log)

	commentLimit := rl(rlKeyCommentUser, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.CommentPerMinute) }, userIDKeyFunc)
	ratingLimit := rl(rlKeyRatingUser, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.RatingPerMinute) }, userIDKeyFunc)
	challengeVisibility := restapimiddleware.VisibilityGuard(deps.Admin.CompetitionParamUC, "challenge_visibility")

	r.Group(func(comments chi.Router) {
		comments.Use(restapimiddleware.CompetitionEnded(competitionUC))
		comments.Use(challengeVisibility)
		comments.Use(requireVerified)
		comments.Use(notBanned)
		comments.Use(commentLimit)
		comments.Get("/challenges/{challengeID}/comments", wrapper.GetChallengesChallengeIDComments)
		comments.Post("/challenges/{challengeID}/comments", wrapper.PostChallengesChallengeIDComments)
		comments.Delete("/comments/{ID}", wrapper.DeleteCommentsID)
	})

	r.Group(func(ratings chi.Router) {
		ratings.Use(restapimiddleware.ChallengeVisibility(competitionUC))
		ratings.Use(challengeVisibility)
		ratings.Use(requireVerified)
		ratings.Use(notBanned)
		ratings.Use(ratingLimit)
		ratings.Get("/challenges/{challengeID}/ratings", wrapper.GetChallengesChallengeIDRatings)
		ratings.Put("/challenges/{challengeID}/rating", wrapper.PutChallengesChallengeIDRating)
	})

	submitLimitWithAudit := restapimiddleware.SubmitRateLimitWithAudit(
		deps.Infra.RedisClient, rlKeySubmitIP, rlKeySubmitUser, defaultRLWindow,
		rateLimitCache, getter,
		ipKeyFunc(), userIDKeyFunc,
		log,
		deps.Comp.SubmissionUC,
		deps.Infra.RatelimitAuditWG,
	)

	hintUnlockUserLimit := rl(rlKeyHintUnlockUser, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.HintUnlockUserPerMinute) }, userIDKeyFunc)

	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(challengeVisibility)
		sub.Use(requireVerified)
		sub.Use(restapimiddleware.RequireTeam())
		sub.Use(notBanned)
		sub.Use(submitLimitWithAudit)
		sub.Post("/challenges/{challengeID}/submit", wrapper.PostChallengesChallengeIDSubmit)
	})

	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(challengeVisibility)
		sub.Use(requireVerified)
		sub.Use(restapimiddleware.RequireTeam())
		sub.Use(notBanned)
		sub.Use(hintUnlockUserLimit)
		sub.Post("/challenges/{challengeID}/hints/{hintID}/unlock", wrapper.PostChallengesChallengeIDHintsHintIDUnlock)
	})
}
