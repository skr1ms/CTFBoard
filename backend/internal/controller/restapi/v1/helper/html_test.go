package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderTrustedHTMLSetsHTMLSecurityHeaders(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	RenderTrustedHTML(rec, http.StatusOK, "<!doctype html><title>ok</title>")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'none'")
	assert.Contains(t, rec.Body.String(), "<title>ok</title>")
}
