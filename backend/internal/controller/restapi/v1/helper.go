package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

func RequireUser(w http.ResponseWriter, r *http.Request) (*entity.User, bool) {
	user, ok := middleware.GetUser(r.Context())
	if !ok {
		httputil.HandleError(w, r, entityError.ErrNotAuthenticated)
		return nil, false
	}
	return user, true
}
