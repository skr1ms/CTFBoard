package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

func TestLogger_CallsNext(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)
	log.EXPECT().WithFields(mock.Anything).Return(child).Maybe()
	child.EXPECT().Info(mock.Anything).Maybe()

	r := chi.NewRouter()
	r.Use(Logger(log, nil))
	called := false
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestLogger_LogsInfo_OnSuccess(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)

	var capturedFields logger.Fields
	log.EXPECT().
		WithFields(mock.Anything).
		Run(func(fields logger.Fields) { capturedFields = fields }).
		Return(child).
		Once()
	child.EXPECT().Info("http request").Once()

	r := chi.NewRouter()
	r.Use(Logger(log, nil))
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.NotNil(t, capturedFields)
	assert.Equal(t, http.StatusOK, capturedFields["status"])
	assert.Equal(t, "GET", capturedFields["method"])
	assert.Equal(t, "/health", capturedFields["path"])
	assert.Equal(t, "192.168.1.1", capturedFields["ip"])
	assert.Equal(t, "test-agent", capturedFields["user_agent"])
	assert.Contains(t, capturedFields, "latency_ms")
	assert.Contains(t, capturedFields, "bytes")
}

func TestLogger_LogsWarn_On4xx(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)

	var capturedFields logger.Fields
	log.EXPECT().
		WithFields(mock.Anything).
		Run(func(fields logger.Fields) { capturedFields = fields }).
		Return(child).
		Once()
	child.EXPECT().Warn("http request error").Once()

	r := chi.NewRouter()
	r.Use(Logger(log, nil))
	r.Get("/forbidden", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.NotNil(t, capturedFields)
	assert.Equal(t, http.StatusForbidden, capturedFields["status"])
}

func TestLogger_LogsError_On5xx(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)

	var capturedFields logger.Fields
	log.EXPECT().
		WithFields(mock.Anything).
		Run(func(fields logger.Fields) { capturedFields = fields }).
		Return(child).
		Once()
	child.EXPECT().Error("http request failed").Once()

	r := chi.NewRouter()
	r.Use(Logger(log, nil))
	r.Get("/broken", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/broken", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.NotNil(t, capturedFields)
	assert.Equal(t, http.StatusInternalServerError, capturedFields["status"])
}

func TestLogger_IncludesQueryAndRequestID_WhenSet(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)

	var capturedFields logger.Fields
	log.EXPECT().
		WithFields(mock.Anything).
		Run(func(fields logger.Fields) { capturedFields = fields }).
		Return(child).
		Once()
	child.EXPECT().Info("http request").Once()

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(Logger(log, nil))
	r.Get("/search", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.NotNil(t, capturedFields)
	assert.Equal(t, "q=test&page=1", capturedFields["query"])
	assert.Contains(t, capturedFields, "request_id")
	assert.NotEmpty(t, capturedFields["request_id"])
}

func TestLogger_RedactsSensitiveQueryParams(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)

	var capturedFields logger.Fields
	log.EXPECT().
		WithFields(mock.Anything).
		Run(func(fields logger.Fields) { capturedFields = fields }).
		Return(child).
		Once()
	child.EXPECT().Info("http request").Once()

	r := chi.NewRouter()
	r.Use(Logger(log, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/?token=secret&page=1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.NotNil(t, capturedFields)
	query, ok := capturedFields["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "REDACTED")
	assert.NotContains(t, query, "secret")
}

func TestLogger_DoesNotRedactSubstringParamName(t *testing.T) {
	t.Parallel()
	log := mocks.NewMockLogger(t)
	child := mocks.NewMockLogger(t)

	var capturedFields logger.Fields
	log.EXPECT().
		WithFields(mock.Anything).
		Run(func(fields logger.Fields) { capturedFields = fields }).
		Return(child).
		Once()
	child.EXPECT().Info("http request").Once()

	r := chi.NewRouter()
	r.Use(Logger(log, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/?mytokenvalue=foo", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.NotNil(t, capturedFields)
	query, ok := capturedFields["query"].(string)
	require.True(t, ok)
	assert.Equal(t, "mytokenvalue=foo", query)
}
