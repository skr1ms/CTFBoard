package e2e_test

import (
	"net/http"
	"testing"
)

// POST /challenges/{ID}/submit with is_regex flag: invalid pattern 400; valid pattern 200; duplicate 409 with ALREADY_SOLVED.
func TestEncryptedRegex_Challenge(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_enc_regex")

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Regex Challenge",
		"description": "Find the pattern",
		"flag":        "CTF{[0-9]{4}}",
		"points":      100,
		"category":    "crypto",
		"is_regex":    true,
		"is_hidden":   false,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("user_enc_regex")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challID, "CTF{abcd}", http.StatusBadRequest).
		Value("error").IsEqual("invalid flag")

	h.SubmitFlag(tokenUser, challID, "CTF{1234}", http.StatusOK).
		Value("message").IsEqual("flag accepted")

	h.SubmitFlag(tokenUser, challID, "CTF{5678}", http.StatusConflict).
		Value("code").IsEqual("ALREADY_SOLVED")
}
