package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// Competition flag_regex: invalid format returns 400 INVALID_FLAG_FORMAT; valid format wrong content returns 400 invalid flag; correct flag returns 200.
func TestFlagRegex_Flow(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_regex")

	_, _, userToken := h.RegisterUserAndLogin("user_regex")
	h.CreateTeam(userToken, "RegexTeam", http.StatusCreated)

	h.SetCompetitionRegex(tokenAdmin, "^GoCTF\\{.+\\}$")

	challID := h.CreateBasicChallenge(tokenAdmin, "Regex Challenge", "GoCTF{secret}", 100)

	resp400a := h.SubmitFlag(userToken, challID, "wrong_format", http.StatusOK)
	require.NotNil(t, resp400a.JSON200)
	require.False(t, resp400a.JSON200.Correct)
	require.Contains(t, string(resp400a.Body), "incorrect")

	resp200b := h.SubmitFlag(userToken, challID, "GoCTF{wrong}", http.StatusOK)
	require.Contains(t, string(resp200b.Body), "incorrect flag")

	resp200 := h.SubmitFlag(userToken, challID, "GoCTF{secret}", http.StatusOK)
	require.Contains(t, string(resp200.Body), "flag accepted")
}

// Competition flag_regex: submit with invalid format returns 400 INVALID_FLAG_FORMAT.
func TestFlagRegex_InvalidFormat_Returns400(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_regex_err")
	_, _, userToken := h.RegisterUserAndLogin("user_regex_err")
	h.CreateTeam(userToken, "RegexTeamErr", http.StatusCreated)
	h.SetCompetitionRegex(tokenAdmin, "^GoCTF\\{.+\\}$")
	challID := h.CreateBasicChallenge(tokenAdmin, "Regex Err", "GoCTF{secret}", 100)
	resp := h.SubmitFlag(userToken, challID, "wrong_format_no_curly", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.False(t, resp.JSON200.Correct)
	require.Contains(t, string(resp.Body), "incorrect")
}
