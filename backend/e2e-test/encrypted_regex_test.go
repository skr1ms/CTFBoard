package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// POST /challenges/{ID}/submit with is_regex flag: invalid pattern 400; valid pattern 200; duplicate 409 with ALREADY_SOLVED.
func TestEncryptedRegex_Challenge(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_enc_regex")
	h.SetCompetitionRegex(tokenAdmin, "^CTF\\{.+\\}$")

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Regex Challenge",
		"description": "Find the pattern",
		"flag":        "CTF{[0-9]+}",
		"points":      100,
		"category":    "crypto",
		"is_regex":    true,
		"is_hidden":   false,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("user_enc_regex")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp400 := h.SubmitFlag(tokenUser, challID, "CTF{abcd}", http.StatusBadRequest)
	require.NotNil(t, resp400.JSON400)
	require.NotNil(t, resp400.JSON400.Error)
	require.Equal(t, "invalid flag", *resp400.JSON400.Error)

	resp200 := h.SubmitFlag(tokenUser, challID, "CTF{1234}", http.StatusOK)
	require.NotNil(t, resp200.JSON200)
	require.Equal(t, "flag accepted", (*resp200.JSON200)["message"])

	resp409 := h.SubmitFlag(tokenUser, challID, "CTF{5678}", http.StatusConflict)
	require.NotNil(t, resp409.JSON409)
	require.NotNil(t, resp409.JSON409.Code)
	require.Equal(t, "ALREADY_SOLVED", *resp409.JSON409.Code)
}

// POST /challenges/{ID}/submit with is_regex: flag not matching pattern returns 400 invalid flag format.
func TestEncryptedRegex_InvalidFlag_Returns400(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)
	_, tokenAdmin := h.SetupCompetition("admin_enc_regex_err")
	h.SetCompetitionRegex(tokenAdmin, "^CTF\\{.+\\}$")
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Regex Err", "description": "x", "flag": "CTF{[0-9]+}",
		"points": 100, "category": "crypto", "is_regex": true, "is_hidden": false,
	})
	_, _, tokenUser := h.RegisterUserAndLogin("user_enc_regex_err")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	resp := h.SubmitFlag(tokenUser, challID, "no-match-pattern", http.StatusBadRequest)
	require.NotNil(t, resp.JSON400)
	require.NotNil(t, resp.JSON400.Error)
	require.Equal(t, "invalid flag format", *resp.JSON400.Error)
}
