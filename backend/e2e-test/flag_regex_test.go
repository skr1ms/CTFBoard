package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Competition flag_regex: invalid format returns 400 INVALID_FLAG_FORMAT; valid format wrong content returns 400 invalid flag; correct flag returns 200.
func TestFlagRegex_Flow(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_regex")

	_, _, userToken := h.RegisterUserAndLogin("user_regex")
	h.CreateTeam(userToken, "RegexTeam", http.StatusCreated)

	h.SetCompetitionRegex(tokenAdmin, "^GoCTF\\{.+\\}$")

	challID := h.CreateBasicChallenge(tokenAdmin, "Regex Challenge", "GoCTF{secret}", 100)

	resp400a := h.SubmitFlag(userToken, challID, "wrong_format", http.StatusBadRequest)
	require.NotNil(t, resp400a.JSON400)
	require.NotNil(t, resp400a.JSON400.Code)
	require.Equal(t, "INVALID_FLAG_FORMAT", *resp400a.JSON400.Code)

	resp400b := h.SubmitFlag(userToken, challID, "GoCTF{wrong}", http.StatusBadRequest)
	require.NotNil(t, resp400b.JSON400)
	require.NotNil(t, resp400b.JSON400.Error)
	require.Equal(t, "invalid flag", *resp400b.JSON400.Error)

	resp200 := h.SubmitFlag(userToken, challID, "GoCTF{secret}", http.StatusOK)
	require.NotNil(t, resp200.JSON200)
	require.Equal(t, "flag accepted", (*resp200.JSON200)["message"])
}

// Competition flag_regex: submit with invalid format returns 400 INVALID_FLAG_FORMAT.
func TestFlagRegex_InvalidFormat_Returns400(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)
	_, tokenAdmin := h.SetupCompetition("admin_regex_err")
	_, _, userToken := h.RegisterUserAndLogin("user_regex_err")
	h.CreateTeam(userToken, "RegexTeamErr", http.StatusCreated)
	h.SetCompetitionRegex(tokenAdmin, "^GoCTF\\{.+\\}$")
	challID := h.CreateBasicChallenge(tokenAdmin, "Regex Err", "GoCTF{secret}", 100)
	resp := h.SubmitFlag(userToken, challID, "wrong_format_no_curly", http.StatusBadRequest)
	require.NotNil(t, resp.JSON400)
	require.NotNil(t, resp.JSON400.Code)
	require.Equal(t, "INVALID_FLAG_FORMAT", *resp.JSON400.Code)
}
