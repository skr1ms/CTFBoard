package httputil

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientIP_NoTrustedProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "1.2.3.4")
	r.RemoteAddr = "5.6.7.8:12345"
	assert.Equal(t, "5.6.7.8", GetClientIP(r, nil))
}

func TestGetClientIP_TrustedProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "1.2.3.4")
	r.RemoteAddr = "127.0.0.1:12345"
	assert.Equal(t, "1.2.3.4", GetClientIP(r, []string{"127.0.0.0/8"}))
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "5.6.7.8:12345"
	assert.Equal(t, "5.6.7.8", GetClientIP(r, nil))
}

func TestGetClientIP_RemoteAddrInvalid(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "no-port"
	assert.Equal(t, "no-port", GetClientIP(r, nil))
}
