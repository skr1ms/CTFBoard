package helper

import (
	"net/http"
	"strconv"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

func RequireUser(w http.ResponseWriter, r *http.Request) (*entity.User, bool) {
	user, ok := middleware.GetUser(r.Context())
	if !ok || user == nil {
		httputil.HandleError(w, r, httperr.ErrNotAuthenticated)
		return nil, false
	}
	return user, true
}

func ClampPage(p *int) int {
	if p == nil || *p < 1 {
		return 1
	}
	return *p
}

func ClampPerPage(p *int, defaultVal, maxVal int) int {
	if p == nil || *p <= 0 {
		return defaultVal
	}
	if *p > maxVal {
		return maxVal
	}
	return *p
}

func ClampLimit(p *int, defaultVal, maxVal int) int {
	if p == nil || *p <= 0 {
		return defaultVal
	}
	if *p > maxVal {
		return maxVal
	}
	return *p
}

func ParseIntQuery(r *http.Request, key string) *int {
	q := r.URL.Query().Get(key)
	if q == "" {
		return nil
	}
	n, err := strconv.Atoi(q)
	if err != nil || n < 0 {
		return nil
	}
	return &n
}

func Ptr[T any](v T) *T {
	return &v
}
