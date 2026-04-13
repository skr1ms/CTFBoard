package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// PUT /challenges/{id}/rating: user without solve gets 403.
func TestRating_Put_RequiresSolve(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_no_solve")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Req Solve", "FLAG{rate_me}", 100)

	_, _, tokenUser := h.RegisterUserAndLogin("rat_nosolve_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.PutRating(tokenUser, challengeID, 4, nil, http.StatusForbidden)
}

// PUT /challenges/{id}/rating: success after solving.
func TestRating_Put_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_put_ok")
	flag := "FLAG{rating_ok}"
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Put OK", flag, 100)

	_, _, tokenUser := h.RegisterUserAndLogin("rat_put_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, flag, http.StatusOK)

	review := "good one"
	resp := h.PutRating(tokenUser, challengeID, 5, &review, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Value)
	assert.Equal(t, 5, *resp.JSON200.Value)
	require.NotNil(t, resp.JSON200.Review)
	assert.Equal(t, review, *resp.JSON200.Review)
}

// PUT /challenges/{id}/rating: updating overwrites previous value.
func TestRating_Put_Update(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_update")
	flag := "FLAG{rating_upd}"
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Update", flag, 100)

	_, _, tokenUser := h.RegisterUserAndLogin("rat_upd_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, flag, http.StatusOK)

	h.PutRating(tokenUser, challengeID, 2, nil, http.StatusOK)
	resp := h.PutRating(tokenUser, challengeID, 4, nil, http.StatusOK)

	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Value)
	assert.Equal(t, 4, *resp.JSON200.Value)
}

// GET /challenges/{id}/ratings: returns ratings list.
func TestRating_Get_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_get")
	flag := "FLAG{rating_get}"
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Get", flag, 100)

	_, _, tokenUser := h.RegisterUserAndLogin("rat_get_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, flag, http.StatusOK)
	h.PutRating(tokenUser, challengeID, 3, nil, http.StatusOK)

	resp := h.GetRatings(tokenUser, challengeID, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.Len(t, *resp.JSON200, 1)
	assert.Equal(t, 3, *(*resp.JSON200)[0].Value)
}

// GET /challenges/{id}/ratings: empty list when no ratings.
func TestRating_Get_Empty(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_get_empty")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Get Empty", "FLAG{empty}", 100)

	_, _, tokenUser := h.RegisterUserAndLogin("rat_empty_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp := h.GetRatings(tokenUser, challengeID, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.Empty(t, *resp.JSON200)
}

// PUT /challenges/{id}/rating: non-existent challenge returns 404.
func TestRating_Put_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_ch_notfound")
	_ = tokenAdmin

	_, _, tokenUser := h.RegisterUserAndLogin("rat_notfound_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.PutRating(tokenUser, uuid.New().String(), 3, nil, http.StatusNotFound)
}

// PUT /challenges/{id}/rating: unauthenticated returns 401.
func TestRating_Put_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_unauth")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Unauth", "FLAG{unauth}", 100)

	h.PutRating("", challengeID, 3, nil, http.StatusUnauthorized)
}

// GET /challenges/{id}/ratings: unauthenticated returns 401.
func TestRating_Get_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("rat_get_unauth")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rating Get Unauth", "FLAG{unauth2}", 100)

	h.GetRatings("", challengeID, http.StatusUnauthorized)
}
