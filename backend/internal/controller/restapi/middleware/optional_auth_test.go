package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestOptionalAuth_BearerHeaderAuthenticates(t *testing.T) {
	t.Parallel()

	svc := newTestJWTService(t)
	userID := uuid.New()
	token, err := svc.GenerateTokenPair(context.Background(), userID, string(domain.RoleUser))
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(OptionalAuth(svc, nil, nil, logkit.Noop()))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, string(domain.RoleUser), GetUserRole(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	})

	ServeAndExpect(t, r, http.MethodGet, "/", map[string]string{"Authorization": "Bearer " + token.AccessToken}, http.StatusNoContent)
}

func TestOptionalAuth_QueryTokenIsIgnored(t *testing.T) {
	t.Parallel()

	svc := newTestJWTService(t)
	userID := uuid.New()
	token, err := svc.GenerateTokenPair(context.Background(), userID, string(domain.RoleUser))
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(OptionalAuth(svc, nil, nil, logkit.Noop()))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, GetUserID(r.Context()))
		assert.Empty(t, GetUserRole(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	})

	ServeAndExpect(t, r, http.MethodGet, "/?token="+token.AccessToken, nil, http.StatusNoContent)
}

func TestOptionalAuth_InvalidOrMissingCredentialsPassThrough(t *testing.T) {
	t.Parallel()

	svc := newTestJWTService(t)

	for _, tt := range []struct {
		name    string
		path    string
		headers map[string]string
	}{
		{name: "missing", path: "/"},
		{name: "invalid header", path: "/", headers: map[string]string{"Authorization": "Bearer invalid-token"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.Use(OptionalAuth(svc, nil, nil, logkit.Noop()))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, GetUserID(r.Context()))
				w.WriteHeader(http.StatusNoContent)
			})

			ServeAndExpect(t, r, http.MethodGet, tt.path, tt.headers, http.StatusNoContent)
		})
	}
}
