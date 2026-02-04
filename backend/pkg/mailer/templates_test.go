package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderVerificationEmail_Success(t *testing.T) {
	data := VerificationData{
		Username:  "user1",
		ActionURL: "https://example.com/verify?token=abc",
		AppName:   "TestApp",
	}
	html, err := RenderVerificationEmail(data, true)
	require.NoError(t, err)
	assert.Contains(t, html, "user1")
	assert.Contains(t, html, "https://example.com/verify?token=abc")
	assert.Contains(t, html, "TestApp")
	assert.Contains(t, html, "Verify Email")
	text, err := RenderVerificationEmail(data, false)
	require.NoError(t, err)
	assert.Contains(t, text, "user1")
	assert.Contains(t, text, "https://example.com/verify?token=abc")
}

func TestRenderVerificationEmail_DefaultAppName(t *testing.T) {
	data := VerificationData{
		Username:  "u",
		ActionURL: "http://x.com",
		AppName:   "",
	}
	html, err := RenderVerificationEmail(data, true)
	require.NoError(t, err)
	assert.Contains(t, html, "CTFBoard")
}

func TestRenderPasswordResetEmail_Success(t *testing.T) {
	data := PasswordResetData{
		Username:  "user1",
		ActionURL: "https://example.com/reset?token=xyz",
		AppName:   "TestApp",
	}
	html, err := RenderPasswordResetEmail(data, true)
	require.NoError(t, err)
	assert.Contains(t, html, "user1")
	assert.Contains(t, html, "https://example.com/reset?token=xyz")
	assert.Contains(t, html, "Reset Password")
	text, err := RenderPasswordResetEmail(data, false)
	require.NoError(t, err)
	assert.Contains(t, text, "user1")
}

func TestRenderPasswordResetEmail_DefaultAppName(t *testing.T) {
	data := PasswordResetData{
		Username:  "u",
		ActionURL: "http://x.com",
		AppName:   "",
	}
	html, err := RenderPasswordResetEmail(data, true)
	require.NoError(t, err)
	assert.Contains(t, html, "CTFBoard")
}
