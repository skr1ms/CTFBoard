package httputil

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "1.2.3.4")
	assert.Equal(t, "1.2.3.4", GetClientIP(r))
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "5.6.7.8:12345"
	assert.Equal(t, "5.6.7.8", GetClientIP(r))
}

func TestGetClientIP_RemoteAddrInvalid(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "no-port"
	assert.Equal(t, "no-port", GetClientIP(r))
}
