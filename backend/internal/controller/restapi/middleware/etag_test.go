package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestETag_AddsETagOnFirstGet(t *testing.T) {
	body := `{"key":"value"}`
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("ETag"))
	assert.Equal(t, body, rec.Body.String())
}

func TestETag_Returns304OnMatchingIfNoneMatch(t *testing.T) {
	body := `{"key":"value"}`
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	// First request to obtain ETag
	req1 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	etag := rec1.Header().Get("ETag")
	require.NotEmpty(t, etag)

	// Second request with If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req2.Header.Set("If-None-Match", etag)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusNotModified, rec2.Code)
	assert.Empty(t, rec2.Body.String())
}

func TestETag_Returns200OnMismatchedIfNoneMatch(t *testing.T) {
	body := `{"key":"value"}`
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("If-None-Match", `"stale-etag"`)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, body, rec.Body.String())
}

func TestETag_PassesThroughNonGetMethods(t *testing.T) {
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code, "method %s", method)
		assert.Empty(t, rec.Header().Get("ETag"), "method %s should have no ETag", method)
	}
}

func TestETag_PassesThroughNon200Responses(t *testing.T) {
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Header().Get("ETag"))
	assert.Equal(t, `{"error":"not found"}`, rec.Body.String())
}

func TestETag_HeadRequestNoBody(t *testing.T) {
	body := `{"key":"value"}`
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequest(http.MethodHead, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("ETag"))
	assert.Empty(t, rec.Body.String())
}

func TestETag_PreservesExistingResponseHeaders(t *testing.T) {
	handler := ETag(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=30")
		w.Header().Set("Vary", "Accept-Encoding")
		_, _ = w.Write([]byte(`{}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "public, max-age=30", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
	assert.NotEmpty(t, rec.Header().Get("ETag"))
}
