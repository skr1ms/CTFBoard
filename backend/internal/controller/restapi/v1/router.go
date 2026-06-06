package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// NewRouter wires the entire HTTP route tree onto router. It creates the Server
// and the OpenAPI wrapper (with typed error handlers for missing/invalid params),
// builds shared infrastructure (cachekit, IP tracking, ban middleware, scoreboard
// visibility), then delegates to the route-group setup functions.
func NewRouter(
	ctx context.Context,
	router chi.Router,
	deps *helper.ServerDeps,
	verifyEmails bool,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
) {
	server := NewServer(deps)
	wrapper := openapi.ServerInterfaceWrapper{
		Handler: server,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			var requiredHeader *openapi.RequiredHeaderError
			if errors.As(err, &requiredHeader) {
				server.OnError(w, r, errmap.NewHTTPError(err, http.StatusUnauthorized, "UNAUTHORIZED"), "OpenAPI", "RequiredHeader")

				return
			}

			var requiredParam *openapi.RequiredParamError
			if errors.As(err, &requiredParam) {
				server.OnError(w, r, errmap.NewHTTPError(err, http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "RequiredParam")

				return
			}

			var invalidParam *openapi.InvalidParamFormatError
			if errors.As(err, &invalidParam) {
				server.OnError(w, r, errmap.NewHTTPError(err, http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "InvalidParamFormat")

				return
			}

			server.OnError(w, r, errmap.NewHTTPError(errors.New("invalid request parameter"), http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "BadRequest")
		},
	}
	sharedCache := cachekit.New(deps.Infra.RedisClient)
	ipTracking := restapimiddleware.IPTracking(ctx, deps.User.TrackingUC, deps.Infra.Logger)
	notUserBanned := restapimiddleware.RequireUserNotBanned()
	scoreboardVis := scoreboardVisibilityMiddleware(deps)
	setupPublicRoutes(router, wrapper, deps, deps.Infra.RedisClient, deps.Infra.Logger, rateLimitCache)
	setupAuthOnlyRoutes(router, deps, wrapper, sharedCache, rateLimitCache, notUserBanned)
	setupProtectedRoutes(router, server, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, ipTracking, notUserBanned, scoreboardVis)
}
