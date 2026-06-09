package request

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestEmailRequestsNormalizeRateLimitKeys(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "user@example.com", ForgotPasswordRequestToParams(&openapi.ForgotPasswordRequest{
		Email: " User@Example.COM ",
	}))
	assert.Equal(t, "user@example.com", ResendVerificationByEmailRequestToParams(&openapi.ResendVerificationByEmailRequest{
		Email: " User@Example.COM ",
	}))
}
