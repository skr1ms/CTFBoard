package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
}

func newRequest(method, target string, body io.Reader) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), method, target, body)
}

func ServeAndExpect(t *testing.T, handler http.Handler, method, path string, headers map[string]string, expectStatus int) *httptest.ResponseRecorder {
	t.Helper()

	req := newRequest(method, path, http.NoBody)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, expectStatus, rr.Code)

	return rr
}
