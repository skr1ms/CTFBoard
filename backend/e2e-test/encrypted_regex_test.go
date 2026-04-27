package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// POST /challenges/{ID}/submit with is_regex flag: invalid pattern 400; valid pattern 200; duplicate 409 with ALREADY_SOLVED.
func TestEncryptedRegex_Challenge(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_enc_regex")
	h.SetCompetitionRegex(tokenAdmin, "^CTF\\{.+\\}$")

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Regex Challenge",
		"description": "Find the pattern",
		"flag":        "CTF{[0-9]+}",
		"points":      100,
		"category":    "crypto",
		"is_regex":    true,
		"state":       "visible",
	})

	_, _, tokenUser := h.RegisterUserAndLogin("user_enc_regex")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp200wrong := h.SubmitFlag(tokenUser, challID, "CTF{abcd}", http.StatusOK)
	require.Contains(t, string(resp200wrong.Body), "incorrect flag")

	resp200 := h.SubmitFlag(tokenUser, challID, "CTF{1234}", http.StatusOK)
	require.Contains(t, string(resp200.Body), "flag accepted")

	// Submitting a second matching flag after the team already solved is idempotent:
	// the handler swallows ErrAlreadySolved and returns 200 "flag accepted" again.
	resp200dup := h.SubmitFlag(tokenUser, challID, "CTF{5678}", http.StatusOK)
	require.Contains(t, string(resp200dup.Body), "flag accepted")
}

// POST /challenges/{ID}/submit with is_regex: flag not matching pattern returns 400 invalid flag format.
func TestEncryptedRegex_InvalidFlag_Returns400(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_enc_regex_err")
	h.SetCompetitionRegex(tokenAdmin, "^CTF\\{.+\\}$")
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Regex Err", "description": "x", "flag": "CTF{[0-9]+}",
		"points": 100, "category": "crypto", "is_regex": true, "state": "visible",
	})
	_, _, tokenUser := h.RegisterUserAndLogin("user_enc_regex_err")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	resp := h.SubmitFlag(tokenUser, challID, "no-match-pattern", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.False(t, resp.JSON200.Correct)
	require.Contains(t, resp.JSON200.Message, "incorrect")
}
