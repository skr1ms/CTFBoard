package emailtemplate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderVerificationEmail_Success(t *testing.T) {
	data := VerificationData{
		Username:  "alice",
		ActionURL: "https://example.com/verify",
		AppName:   "AstroCTF",
	}

	html, err := RenderVerificationEmail(data, true)
	require.NoError(t, err)
	require.Contains(t, html, "Verify your email address")
	require.Contains(t, html, data.ActionURL)
	require.Contains(t, html, data.AppName)

	text, err := RenderVerificationEmail(data, false)
	require.NoError(t, err)
	require.Contains(t, text, "Verify your email address")
	require.Contains(t, text, data.ActionURL)
	require.NotContains(t, strings.ToLower(text), "<html")
}

func TestRenderPasswordResetEmail_Success(t *testing.T) {
	data := PasswordResetData{
		Username:  "alice",
		ActionURL: "https://example.com/reset",
		AppName:   "AstroCTF",
	}

	html, err := RenderPasswordResetEmail(data, true)
	require.NoError(t, err)
	require.Contains(t, html, "Reset your password")
	require.Contains(t, html, data.ActionURL)

	text, err := RenderPasswordResetEmail(data, false)
	require.NoError(t, err)
	require.Contains(t, text, "Reset your password")
	require.Contains(t, text, data.ActionURL)
	require.NotContains(t, strings.ToLower(text), "<html")
}
