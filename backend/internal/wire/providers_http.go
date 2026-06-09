package wire

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/setup"
)

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
		router.Use(cors.Handler(corsOptions(cfg)))
	}

	defaultTimeoutMW := kitMiddleware.Timeout(requestTimeout)
	longTimeoutMW := kitMiddleware.Timeout(longRequestTimeout)

	router.Use(func(next http.Handler) http.Handler {
		withDefaultTimeout := defaultTimeoutMW(next)
		withLongTimeout := longTimeoutMW(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/ws") || strings.HasSuffix(r.URL.Path, "/sse") {
				next.ServeHTTP(w, r)

				return
			}

			if isLongRequestPath(r.URL.Path) {
				withLongTimeout.ServeHTTP(w, r)

				return
			}

			withDefaultTimeout.ServeHTTP(w, r)
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

	setupUC := setup.NewSetupUseCase(setup.SetupDeps{
		UserUC:      deps.User.UserUC,
		CompUC:      deps.Comp.CompetitionUC,
		CompParamUC: deps.Admin.CompetitionParamUC,
		SettingsUC:  deps.Admin.SettingsUC,
		TM:          deps.Infra.TM,
		JWTService:  deps.Infra.JWTService,
	})
	setupHandler := v1.NewSetupHandler(setupUC, l, deps.Infra.Validator, cfg.SetupToken, cfg.SecureCookies, int(cfg.RefreshTTL.Seconds()))

	// Paths that remain accessible before setup is complete.
	setupAllowlist := restapimiddleware.SetupAllowlist{
		Exact: []string{
			"/api/v1/setup",
			"/api/v1/configs/public",
			"/api/v1/health",
			"/api/v1/healthcheck",
			"/health",
			"/metrics",
		},
		Prefixes: []string{
			"/api/v1/avatars/",
		},
	}
	setupRequiredMW := restapimiddleware.SetupRequired(setupUC, setupAllowlist)

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

func isLongRequestPath(path string) bool {
	if path == "/api/v1/admin/import" ||
		path == "/api/v1/admin/import/csv" ||
		path == "/api/v1/admin/export/zip" {
		return true
	}

	if strings.HasPrefix(path, "/api/v1/files/download/") {
		return true
	}

	return strings.HasPrefix(path, "/api/v1/admin/challenges/") && strings.HasSuffix(path, "/files")
}

func corsOptions(cfg *config.Config) cors.Options {
	return cors.Options{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-Setup-Token",
		},
		ExposedHeaders: []string{
			"Link",
			"Retry-After",
		},
		AllowCredentials: true,
		MaxAge:           corsPreflightMaxAgeSeconds,
	}
}
