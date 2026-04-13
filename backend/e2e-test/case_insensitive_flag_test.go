package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// TestCaseInsensitiveFlag_PlainFlag_MatchesDifferentCase verifies that a flag stored in
// mixed case is accepted when submitted in a different case, provided is_case_insensitive=true.
func TestCaseInsensitiveFlag_PlainFlag_MatchesDifferentCase(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("ci_flag_" + suffix)

	storedFlag := "flag{CasE_TeSt_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":               "CaseInsensitive_" + suffix,
		"description":         "case insensitive flag test",
		"flag":                storedFlag,
		"points":              100,
		"category":            "misc",
		"state":               "visible",
		"is_case_insensitive": true,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("ci_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Submit the flag in ALL UPPERCASE - must be accepted as correct.
	upperFlag := "FLAG{CASE_TEST_" + suffix + "}"
	resp := h.SubmitFlag(tokenUser, challID, upperFlag, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.True(t, resp.JSON200.Correct, "expected correct=true for case-insensitive match")
}

// TestCaseInsensitiveFlag_Disabled_RejectsDifferentCase verifies that a differently-cased flag
// is treated as incorrect when is_case_insensitive=false (the default).
func TestCaseInsensitiveFlag_Disabled_RejectsDifferentCase(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("ci_off_" + suffix)

	storedFlag := "flag{CasE_TeSt_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":               "CaseSensitive_" + suffix,
		"description":         "case sensitive flag test",
		"flag":                storedFlag,
		"points":              100,
		"category":            "misc",
		"state":               "visible",
		"is_case_insensitive": false,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("ci_off_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Submit in different case - server returns 200 but correct=false.
	upperFlag := "FLAG{CASE_TEST_" + suffix + "}"
	resp := h.SubmitFlag(tokenUser, challID, upperFlag, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.False(t, resp.JSON200.Correct, "expected correct=false for case-sensitive mismatch")
}
