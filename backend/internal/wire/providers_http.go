package wire

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsController "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/setup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/sse"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func ProvideServerDeps(
	ctx context.Context,
	cfg *config.Config,
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
	submissionUC *competition.SubmissionUseCase,
	tagUC *challenge.TagUseCase,
	fieldUC *settings.FieldUseCase,
	pageUC *page.PageUseCase,
	bracketUC *competition.BracketUseCase,
	notifUC *notification.NotificationUseCase,
	apiTokenUC *user.APITokenUseCase,
	backupUC *backup.BackupUseCase,
	settingsUC *settings.SettingsUseCase,
	storageAdminUC usecase.StorageAdminUseCase,
	competitionParamUC *competition.CompetitionParamUseCase,
	commentUC *challenge.CommentUseCase,
	ratingUC *challenge.RatingUseCase,
	trackingUC *user.TrackingUseCase,
	oauthUC *user.OAuthUseCase,
	avatarUC *avatar.AvatarUseCase,
	appealUC *user.BanAppealUseCase,
	jwtService *jwtkit.JWTService,
	redisClient *redis.Client,
	wsCtrl *wsController.Controller,
	wsHub *wskit.Hub,
	v validator.Validator,
	runtimeInvalidator *runtimeSettingsInvalidator,
	l logkit.Logger,
) (*helper.ServerDeps, error) {
	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyForgot, forgotPasswordRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create forgot-password rate limiter: %w", err)
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResend, resendVerificationRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create resend-verification rate limiter: %w", err)
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResetTok, resetPasswordTokenRateLimit, resetPasswordTokenRateWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create reset-password-token rate limiter: %w", err)
	}

	rateLimitCache := restapimiddleware.NewRateLimitConfigCache(ctx, rateLimitCacheTTL)
	runtimeInvalidator.SetRateLimitCache(rateLimitCache)

	ratelimitAuditWG := &sync.WaitGroup{}

	return &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ChallengeUC: challengeUC,
			HintUC:      hintUC,
			FileUC:      fileUC,
			TagUC:       tagUC,
			CommentUC:   commentUC,
			RatingUC:    ratingUC,
		},
		Team: helper.TeamDeps{
			TeamUC:  teamUC,
			AwardUC: awardUC,
		},
		User: helper.UserDeps{
			UserUC:              userUC,
			EmailUC:             emailUC,
			APITokenUC:          apiTokenUC,
			TrackingUC:          trackingUC,
			OAuthUC:             oauthUC,
			AvatarUC:            avatarUC,
			AppealUC:            appealUC,
			FrontendURL:         cfg.FrontendURL,
			SecureCookies:       cfg.SecureCookies,
			RefreshCookieMaxAge: int(cfg.RefreshTTL.Seconds()),
			OAuthGitHubEnabled:  cfg.GitHub.IsConfigured(),
			OAuthGoogleEnabled:  cfg.Google.IsConfigured(),
		},
		Comp: helper.CompetitionDeps{
			CompetitionUC: competitionUC,
			SolveUC:       solveUC,
			StatsUC:       statsUC,
			SubmissionUC:  submissionUC,
			BracketUC:     bracketUC,
		},
		Admin: helper.AdminDeps{
			BackupUC:           backupUC,
			SettingsUC:         settingsUC,
			CompetitionParamUC: competitionParamUC,
			StorageAdminUC:     storageAdminUC,
			FieldUC:            fieldUC,
			PageUC:             pageUC,
			NotifUC:            notifUC,
		},
		Infra: helper.InfraDeps{
			JWTService:                    jwtService,
			RedisClient:                   redisClient,
			WSController:                  wsCtrl,
			SSEHandler:                    sse.NewSSEHandler(wsHub, l),
			Validator:                     v,
			Logger:                        l,
			TrustedProxyCIDRs:             cfg.TrustedProxyCIDRs,
			StructuredLogger:              cfg.StructuredLogger,
			DebugEnabled:                  cfg.DebugEnabled,
			RateLimitConfigCache:          rateLimitCache,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
			RatelimitAuditWG:              ratelimitAuditWG,
		},
	}, nil
}

func ProvideRouter(
	ctx context.Context,
	cfg *config.Config,
	l logkit.Logger,
	deps *helper.ServerDeps,
	storageProvider storage.Provider,
	runtimeInvalidator *runtimeSettingsInvalidator,
) (chi.Router, error) {
	router := chi.NewRouter()
	router.Use(kitMiddleware.RequestID())

	clientIP, err := kitMiddleware.ClientIP(cfg.TrustedProxyCIDRs)
	if err != nil {
		return nil, fmt.Errorf("ProvideRouter - ClientIP: %w", err)
	}

	router.Use(clientIP)

	if cfg.StructuredLogger {
		router.Use(kitMiddleware.Logger(l, cfg.TrustedProxyCIDRs))
	} else {
		router.Use(middleware.Logger)
	}

	router.Use(kitMiddleware.Metrics(prometheus.DefaultRegisterer, httputil.ChiPathFromRequest))
	router.Use(kitMiddleware.Recoverer(l))

	// CORS must be registered before any rate-limit middleware so that 429
	// responses (short-circuited by httprate.WithLimitHandler) still carry
	// Access-Control-Allow-Origin headers and the browser doesn't report a
	// spurious CORS error instead of a rate-limit error.
	if len(cfg.CORSOrigins) > 0 {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           corsPreflightMaxAgeSeconds,
		}))
	}

	timeoutMW := kitMiddleware.Timeout(requestTimeout)

	router.Use(func(next http.Handler) http.Handler {
		withTimeout := timeoutMW(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/ws") || strings.HasSuffix(r.URL.Path, "/sse") {
				next.ServeHTTP(w, r)

				return
			}

			withTimeout.ServeHTTP(w, r)
		})
	})

	strictSecurity := kitMiddleware.SecurityHeaders(true)
	swaggerDocCSP := "default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'"
	relaxedSecurity := kitMiddleware.SecurityHeaders(true, kitMiddleware.WithCSP(swaggerDocCSP))

	router.Use(func(next http.Handler) http.Handler {
		strictH := strictSecurity(next)
		relaxedH := relaxedSecurity(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSwaggerOrOpenAPIDocPath(r.URL.Path) {
				relaxedH.ServeHTTP(w, r)

				return
			}

			strictH.ServeHTTP(w, r)
		})
	})

	generalIPLimitMiddleware := restapimiddleware.DynamicRateLimit(
		deps.Infra.RedisClient, rlKeyGeneral, time.Minute,
		deps.Infra.RateLimitConfigCache, deps.Admin.SettingsUC,
		func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		func(r *http.Request) (string, error) {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		},
		l,
	)
	router.Use(generalIPLimitMiddleware)

	healthHandler := httputil.HealthHandler(map[string]httputil.Checker{
		"db": healthCheckerFunc(func(ctx context.Context) error {
			_, err := deps.Admin.SettingsUC.Get(ctx)

			return err
		}),
		"redis":   healthCheckerFunc(func(ctx context.Context) error { return deps.Infra.RedisClient.Ping(ctx).Err() }),
		"storage": healthCheckerFunc(func(ctx context.Context) error { return storageProvider.Ping(ctx) }),
	})
	router.Get("/health", healthHandler)

	metricsHandler := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	)

	metricsHandler = metricsAllowlistMiddleware(cfg.MetricsAllowedIPs, metricsHandler)

	router.Handle("/metrics", metricsHandler)

	openapiJSONHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		swagger, err := openapi.GetSwagger()
		if err != nil {
			httputil.HandleError(w, r, err)

			return
		}

		httputil.RenderOK(w, r, swagger)
	})
	router.Get("/openapi.json", openapiJSONHandler)
	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/openapi.json")))

	deps.Infra.ScoreboardVisibilityCache = restapimiddleware.NewScoreboardVisibilityCache(ctx)
	runtimeInvalidator.SetScoreboardVisibilityCache(deps.Infra.ScoreboardVisibilityCache)

	setupUC := setup.NewSetupUseCase(setup.SetupDeps{
		UserUC:      deps.User.UserUC,
		CompUC:      deps.Comp.CompetitionUC,
		CompParamUC: deps.Admin.CompetitionParamUC,
		SettingsUC:  deps.Admin.SettingsUC,
		JWTService:  deps.Infra.JWTService,
	})
	setupHandler := v1.NewSetupHandler(setupUC, l, deps.Infra.Validator, cfg.SetupToken, cfg.SecureCookies, int(cfg.RefreshTTL.Seconds()))

	// Paths that remain accessible before setup is complete.
	setupAllowedPaths := []string{
		"/api/v1/setup",
		"/api/v1/configs/public",
		"/api/v1/health",
		"/api/v1/healthcheck",
		"/health",
		"/metrics",
		"/avatars/",
	}
	setupRequiredMW := restapimiddleware.SetupRequired(setupUC, setupAllowedPaths)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler)
		r.Handle("/metrics", metricsHandler)
		r.Get("/openapi.json", openapiJSONHandler)
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/api/v1/openapi.json")))

		// Setup endpoints - always accessible, outside the SetupRequired gate.
		r.Get("/setup/status", setupHandler.GetSetupStatus)
		r.Post("/setup", setupHandler.PostSetup)

		// All other API routes gated by setup completion.
		r.Group(func(gated chi.Router) {
			gated.Use(setupRequiredMW)
			v1.NewRouter(ctx, gated, deps, cfg.VerifyEmails, deps.Infra.RateLimitConfigCache)
		})
	})

	return router, nil
}

func metricsAllowlistMiddleware(allowedIPs []string, next http.Handler) http.Handler {
	nets := make([]*net.IPNet, 0, len(allowedIPs))

	ips := make([]net.IP, 0, len(allowedIPs))
	for _, s := range allowedIPs {
		if strings.Contains(s, "/") {
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				continue
			}

			nets = append(nets, n)
		} else {
			ip := net.ParseIP(s)
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

		ip := net.ParseIP(clientIP)
		if ip == nil {
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))

			return
		}

		for _, n := range nets {
			if n.Contains(ip) {
				next.ServeHTTP(w, r)

				return
			}
		}

		if slices.ContainsFunc(ips, ip.Equal) {
			next.ServeHTTP(w, r)

			return
		}

		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))
	})
}

func isSwaggerOrOpenAPIDocPath(path string) bool {
	switch path {
	case "/openapi.json", "/api/v1/openapi.json", "/swagger", "/api/v1/swagger":
		return true
	default:
		return strings.HasPrefix(path, "/swagger/") || strings.HasPrefix(path, "/api/v1/swagger/")
	}
}

func ProvideServer(router chi.Router, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}
}

func ProvideApp(server *http.Server, userRepo repo.UserRepository, solveUC *competition.SolveUseCase, avatarUC *avatar.AvatarUseCase, serverDeps *helper.ServerDeps, broadcaster *iws.Broadcaster) *App {
	return &App{
		Server:           server,
		UserRepo:         userRepo,
		SolveUseCase:     solveUC,
		AvatarUC:         avatarUC,
		RatelimitAuditWG: serverDeps.Infra.RatelimitAuditWG,
		Broadcaster:      broadcaster,
	}
}
