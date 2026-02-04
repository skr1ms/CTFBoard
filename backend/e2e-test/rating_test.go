package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /admin/ctf-events + GET /ratings: endpoints respond successfully.
func TestRating_CTFEventsAndRatings_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_ratings_ok")

	now := time.Now().UTC()
	suffix := uuid.New().String()[:8]
	evName := "event_" + suffix

	createResp := h.CreateCTFEvent(tokenAdmin, evName, now.Add(-2*time.Hour), now.Add(-1*time.Hour), 1.0, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	listResp := h.GetAdminCTFEvents(tokenAdmin, http.StatusOK)
	require.NotNil(t, listResp.JSON200)

	h.GetRatings(1, 50, http.StatusOK)
}

// POST /admin/ctf-events: non-admin gets 403.
func TestRating_CreateCTFEvent_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("rating_user_" + suffix)

	now := time.Now().UTC()
	h.CreateCTFEvent(tokenUser, "x", now, now, 1.0, http.StatusForbidden)
}

// GET /ratings/team/{id}: returns team ratings (may be empty).
func TestRating_GetTeamRatings_NoRating_Returns404(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("rating_team_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)

	h.GetTeamRatings(*team.JSON200.ID, http.StatusNotFound)
}

// POST /admin/ctf-events/{id}/finalize: non-admin gets 403.
func TestRating_Finalize_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_rating_fin")
	now := time.Now().UTC()
	suffix := uuid.New().String()[:8]
	createResp := h.CreateCTFEvent(tokenAdmin, "ev_"+suffix, now.Add(-2*time.Hour), now.Add(-1*time.Hour), 1.0, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	_, _, tokenUser := h.RegisterUserAndLogin("rating_fin_user_" + suffix)

	h.FinalizeCTFEvent(tokenUser, *createResp.JSON201.ID, http.StatusForbidden)
}
