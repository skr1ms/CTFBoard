package e2e_test

import (
	"net/http"
	"testing"
)

func TestFlagRegex_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition using helper (Register Admin & Start)
	_, tokenAdmin := h.SetupCompetition("admin_regex")

	// 2. Register Regular User and Create Team
	_, _, userToken := h.RegisterUserAndLogin("user_regex")
	h.CreateTeam(userToken, "RegexTeam", http.StatusCreated)

	// 3. Configure Competition with Flag Regex
	h.SetCompetitionRegex(tokenAdmin, "^GoCTF\\{.+\\}$")

	// 4. Create Challenge with a valid flag (matching the regex)
	challID := h.CreateBasicChallenge(tokenAdmin, "Regex Challenge", "GoCTF{secret}", 100)

	// 5. User submits flag with INVALID FORMAT (Expect 400 with INVALID_FLAG_FORMAT code)
	h.SubmitFlag(userToken, challID, "wrong_format", http.StatusBadRequest).
		Value("code").IsEqual("INVALID_FLAG_FORMAT")

	// 6. User submits flag with VALID FORMAT but WRONG CONTENT (Expect 400 with 'invalid flag' error)
	h.SubmitFlag(userToken, challID, "GoCTF{wrong}", http.StatusBadRequest).
		Value("error").IsEqual("invalid flag")

	// 7. User submits flag with VALID FORMAT and CORRECT CONTENT (Expect 200)
	h.SubmitFlag(userToken, challID, "GoCTF{secret}", http.StatusOK).
		Value("message").IsEqual("flag accepted")
}
