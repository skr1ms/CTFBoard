package e2e_test

import (
	"net/http"
	"testing"
)

// Competition flag_regex: invalid format returns 400 INVALID_FLAG_FORMAT; valid format wrong content returns 400 invalid flag; correct flag returns 200.
func TestFlagRegex_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_regex")

	_, _, userToken := h.RegisterUserAndLogin("user_regex")
	h.CreateTeam(userToken, "RegexTeam", http.StatusCreated)

	h.SetCompetitionRegex(tokenAdmin, "^GoCTF\\{.+\\}$")

	challID := h.CreateBasicChallenge(tokenAdmin, "Regex Challenge", "GoCTF{secret}", 100)

	h.SubmitFlag(userToken, challID, "wrong_format", http.StatusBadRequest).
		Value("code").IsEqual("INVALID_FLAG_FORMAT")

	h.SubmitFlag(userToken, challID, "GoCTF{wrong}", http.StatusBadRequest).
		Value("error").IsEqual("invalid flag")

	h.SubmitFlag(userToken, challID, "GoCTF{secret}", http.StatusOK).
		Value("message").IsEqual("flag accepted")
}
