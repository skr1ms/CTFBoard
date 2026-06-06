package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewS3HTTPTransport_ConfiguresTimeouts(t *testing.T) {
	t.Parallel()

	transport := newS3HTTPTransport()

	assert.NotNil(t, transport.Proxy)
	assert.NotNil(t, transport.DialContext)
	assert.Equal(t, s3MaxIdleConns, transport.MaxIdleConns)
	assert.Equal(t, s3MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	assert.Equal(t, s3TLSHandshakeTimeout, transport.TLSHandshakeTimeout)
	assert.Equal(t, s3ResponseHeaderTimeout, transport.ResponseHeaderTimeout)
	assert.Equal(t, s3IdleConnTimeout, transport.IdleConnTimeout)
	assert.Equal(t, s3ExpectContinueTimeout, transport.ExpectContinueTimeout)
}
