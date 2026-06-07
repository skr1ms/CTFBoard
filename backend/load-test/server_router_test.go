package load_test

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	v1helper "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func buildLoadTestRouter(ctx context.Context, l logkit.Logger, uc *loadTestUseCases, val validator.Validator, jwtSvc *jwtkit.JWTService, storageDir string, redisClient *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	clientIP, err := kitMiddleware.ClientIP(nil)
	if err != nil {
		panic(err)
	}

	r.Use(kitMiddleware.RequestID(), clientIP, kitMiddleware.Recoverer(l))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/ws") {
				next.ServeHTTP(w, r)

				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, "lt:forgot", 100000, 24*time.Hour)
	if err != nil {
		panic("load-test: failed to create forgot-password rate limiter: " + err.Error())
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, "lt:resend", 100000, 24*time.Hour)
	if err != nil {
		panic("load-test: failed to create resend-verification rate limiter: " + err.Error())
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, "lt:reset-token", 100000, time.Minute)
	if err != nil {
		panic("load-test: failed to create reset-password-token rate limiter: " + err.Error())
	}

	deps := &v1helper.ServerDeps{
		Challenge: v1helper.ChallengeDeps{
			ReadUC: uc.challenge, SubmitUC: uc.challenge, AdminUC: uc.challenge,
			HintUC: uc.hint, FileUC: uc.file, TagUC: uc.tagUC, CommentUC: uc.commentUC,
		},
		Team:  v1helper.TeamDeps{ReadUC: uc.team, SelfUC: uc.team, AdminUC: uc.team, AwardUC: uc.award},
		User:  v1helper.UserDeps{UserUC: uc.user, EmailUC: uc.email, APITokenUC: uc.apiTokenUC, TrackingUC: uc.trackingUC},
		Comp:  v1helper.CompetitionDeps{CompetitionUC: uc.competition, SolveUC: uc.solve, StatsUC: uc.stats, SubmissionUC: uc.submissionUC, BracketUC: uc.bracketUC},
		Admin: v1helper.AdminDeps{BackupUC: uc.backup, SettingsUC: uc.settings, CompetitionParamUC: uc.competitionParamUC, FieldUC: uc.fieldUC, PageUC: uc.pageUC, NotifUC: uc.notifUC},
		Infra: v1helper.InfraDeps{
			JWTService:                    jwtSvc,
			RedisClient:                   redisClient,
			WSController:                  uc.ws,
			Validator:                     val,
			Logger:                        l,
			TrustedProxyCIDRs:             nil,
			StructuredLogger:              false,
			DebugEnabled:                  false,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
		},
	}

	r.Route("/api/v1", func(apiRouter chi.Router) {
		rateLimitCache := restapimiddleware.NewRateLimitConfigCache(context.Background(), 30*time.Second)
		v1.NewRouter(ctx, apiRouter, deps, false, rateLimitCache)

		apiRouter.Get("/files/download/*", func(w http.ResponseWriter, r *http.Request) {
			fs := http.StripPrefix("/api/v1/files/download/", http.FileServer(http.Dir(storageDir)))
			fs.ServeHTTP(w, r)
		})
	})

	return r
}
