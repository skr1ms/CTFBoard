package e2e_test

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	appwire "github.com/TakuyaYagam1/AstroCTFb/internal/wire"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const testProductionSetupToken = "setup-token-12345678901234567890"

// Router (chi, middleware, api v1 routes).
func setupTestRouter(ctx context.Context, l logkit.Logger, uc *testUseCases, validatorService validator.Validator, jwtService *jwtkit.JWTService, _ string) *chi.Mux {
	r := chi.NewRouter()
	timeoutMW := kitMiddleware.Timeout(60 * time.Second)

	clientIP, err := kitMiddleware.ClientIP(nil)
	if err != nil {
		panic(err)
	}

	r.Use(kitMiddleware.RequestID(), clientIP, kitMiddleware.Recoverer(l))
	r.Use(func(next http.Handler) http.Handler {
		withTimeout := timeoutMW(next)

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/ws") {
				next.ServeHTTP(w, req)

				return
			}

			withTimeout.ServeHTTP(w, req)
		})
	})
	r.Use(kitMiddleware.Logger(l, nil))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:forgot", 20, 24*time.Hour)
	if err != nil {
		panic("e2e: failed to create forgot-password rate limiter: " + err.Error())
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:resend", 10, 24*time.Hour)
	if err != nil {
		panic("e2e: failed to create resend-verification rate limiter: " + err.Error())
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:reset-token", 20, time.Minute)
	if err != nil {
		panic("e2e: failed to create reset-password-token rate limiter: " + err.Error())
	}

	deps := buildTestServerDeps(ctx, l, uc, validatorService, jwtService, forgotLimiter, resendLimiter, resetTokenLimiter)

	r.Route("/api/v1", func(apiRouter chi.Router) {
		v1.NewRouter(ctx, apiRouter, deps, false, testRateLimitCache)
	})

	return r
}

func setupProductionTestRouter(ctx context.Context, l logkit.Logger, uc *testUseCases, validatorService validator.Validator, jwtService *jwtkit.JWTService, storageProvider storage.Provider) chi.Router {
	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:prod:forgot", 20, 24*time.Hour)
	if err != nil {
		panic("e2e: failed to create production forgot-password rate limiter: " + err.Error())
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:prod:resend", 10, 24*time.Hour)
	if err != nil {
		panic("e2e: failed to create production resend-verification rate limiter: " + err.Error())
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:prod:reset-token", 20, time.Minute)
	if err != nil {
		panic("e2e: failed to create production reset-password-token rate limiter: " + err.Error())
	}

	deps := buildTestServerDeps(ctx, l, uc, validatorService, jwtService, forgotLimiter, resendLimiter, resetTokenLimiter)
	cfg := &config.Config{
		App: config.App{
			SetupToken:       testProductionSetupToken,
			SecureCookies:    false,
			StructuredLogger: false,
		},
		HTTP: config.HTTP{
			Port:        "8080",
			CORSOrigins: []string{"http://localhost:3000"},
		},
		JWT: config.JWT{
			RefreshTTL: 72 * time.Hour,
		},
	}

	router, err := appwire.ProvideRouter(ctx, cfg, l, deps, storageProvider, nil)
	if err != nil {
		panic("e2e: failed to create production router: " + err.Error())
	}

	return router
}

func buildTestServerDeps(
	ctx context.Context,
	l logkit.Logger,
	uc *testUseCases,
	validatorService validator.Validator,
	jwtService *jwtkit.JWTService,
	forgotLimiter *restapimiddleware.PerKeyRateLimiter,
	resendLimiter *restapimiddleware.PerKeyRateLimiter,
	resetTokenLimiter *restapimiddleware.PerKeyRateLimiter,
) *helper.ServerDeps {
	deps := &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ReadUC: uc.challenge, SubmitUC: uc.challenge, AdminUC: uc.challenge, HintUC: uc.hint, FileUC: uc.file, TagUC: uc.tagUC, CommentUC: uc.commentUC, RatingUC: uc.ratingUC,
		},
		Team:  helper.TeamDeps{ReadUC: uc.team, SelfUC: uc.team, AdminUC: uc.team, AwardUC: uc.award},
		User:  helper.UserDeps{UserUC: uc.user, EmailUC: uc.email, APITokenUC: uc.apiTokenUC, TrackingUC: uc.trackingUC, AvatarUC: uc.avatarUC, AppealUC: uc.appealUC},
		Comp:  helper.CompetitionDeps{CompetitionUC: uc.competition, SolveUC: uc.solve, StatsUC: uc.stats, SubmissionUC: uc.submissionUC, BracketUC: uc.bracketUC},
		Admin: helper.AdminDeps{BackupUC: uc.backup, SettingsUC: uc.settings, CompetitionParamUC: uc.competitionParamUC, StorageAdminUC: uc.storageAdminUC, FieldUC: uc.fieldUC, PageUC: uc.pageUC, NotifUC: uc.notifUC},
		Infra: helper.InfraDeps{
			JWTService:                    jwtService,
			RedisClient:                   TestRedis,
			WSController:                  uc.ws,
			Validator:                     validatorService,
			Logger:                        l,
			TrustedProxyCIDRs:             nil,
			StructuredLogger:              false,
			DebugEnabled:                  false,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
			TM:                            uc.TM,
		},
	}
	testRateLimitCache = restapimiddleware.NewRateLimitConfigCache(context.Background(), time.Second)
	deps.Infra.RateLimitConfigCache = testRateLimitCache

	testCompetitionUC = uc.competition

	return deps
}

// Utils

// getEnv returns os.LookupEnv(key) or fallback
