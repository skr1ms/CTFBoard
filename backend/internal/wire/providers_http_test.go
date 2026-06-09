package wire

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/config"
)

func TestCORSOptions_AllowsSetupTokenPreflightAndExposesRetryAfter(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(cors.Handler(corsOptions(&config.Config{
		HTTP: config.HTTP{CORSOrigins: []string{"https://example.com"}},
	})))
	r.Post("/api/v1/setup", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/setup", http.NoBody)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type,x-setup-token")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "X-Setup-Token")
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type")

	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup", http.NoBody)
	req.Header.Set("Origin", "https://example.com")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Access-Control-Expose-Headers"), "Retry-After")
}

func TestIsLongRequestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/v1/admin/import", want: true},
		{path: "/api/v1/admin/import/csv", want: true},
		{path: "/api/v1/admin/export/zip", want: true},
		{path: "/api/v1/files/download/challenges/file.bin", want: true},
		{path: "/api/v1/admin/challenges/018f7c2d-6e0b-75a2-9f5c-0be9890d17aa/files", want: true},
		{path: "/api/v1/admin/challenges/018f7c2d-6e0b-75a2-9f5c-0be9890d17aa/files/1", want: false},
		{path: "/api/v1/challenges", want: false},
		{path: "/api/v1/ws", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, isLongRequestPath(tt.path))
		})
	}
}

func TestProvideServerUsesHeaderReadTimeoutAndLongWriteTimeout(t *testing.T) {
	t.Parallel()

	server := ProvideServer(chi.NewRouter(), &config.Config{HTTP: config.HTTP{Port: "8080"}})

	require.Zero(t, server.ReadTimeout)
	require.Equal(t, 15*time.Second, server.ReadHeaderTimeout)
	require.Equal(t, 10*time.Minute, server.WriteTimeout)
}
