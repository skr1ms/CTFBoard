package load_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestRace_ConcurrentTeamJoin(t *testing.T) {
	require.NotNil(t, Fixture)
	require.GreaterOrEqual(t, len(Fixture.Users), raceConcurrency,
		"need at least %d seeded users for team join race test", raceConcurrency)

	ctx := context.Background()
	client, err := openapi.NewClientWithResponses(Fixture.BaseURL + "/api/v1")
	require.NoError(t, err)

	// Create a host user whose team will be the join target.
	hostIdx := len(Fixture.Users) - 1
	hostToken := Fixture.Users[hostIdx].Token
	hostAuth := bearerEditor(hostToken)

	// Create a named team for the host (replacing their seeded solo team).
	// Solo teams reject join requests, so a named team is required as the join target.
	confirmReset := true
	createResp, err := client.PostTeamsWithResponse(ctx, openapi.CreateTeamRequest{
		Name:         "race-join-target",
		ConfirmReset: &confirmReset,
	}, hostAuth)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(),
		"host must be able to create named team: %s", string(createResp.Body))

	inviteResp, err := client.GetTeamsMeInviteWithResponse(ctx, hostAuth)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, inviteResp.StatusCode())
	require.NotNil(t, inviteResp.JSON200)
	require.NotNil(t, inviteResp.JSON200.InviteToken)

	inviteToken := *inviteResp.JSON200.InviteToken

	// Use users from the middle of the fixture slice as joiners.
	// Users[0:raceConcurrency] are avoided because HintUnlockTargeter starts its round-robin
	// from index 0; after joining a new team those users would have no awards and cause
	// ErrInsufficientPoints (402) in the subsequent stress hint test.
	joinerStart := len(Fixture.Users) / 2
	joiners := Fixture.Users[joinerStart : joinerStart+raceConcurrency]

	var (
		successes atomic.Int32
		errors500 atomic.Int32
		wg        sync.WaitGroup
	)

	start := make(chan struct{})

	for _, u := range joiners {
		wg.Add(1)

		go func(token string) {
			defer wg.Done()

			<-start

			body, _ := json.Marshal(map[string]any{"invite_token": inviteToken, "confirm_reset": true})

			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost,
				Fixture.BaseURL+"/api/v1/teams/join",
				strings.NewReader(string(body)),
			)
			if reqErr != nil {
				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", token)

			resp, doErr := http.DefaultClient.Do(req)
			if doErr != nil {
				return
			}

			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK, http.StatusCreated:
				successes.Add(1)
			case http.StatusInternalServerError:
				errors500.Add(1)
			}
		}(u.Token)
	}

	close(start)
	wg.Wait()

	fmt.Printf("\n[race] Concurrent team join results: ok=%d 500=%d\n",
		successes.Load(), errors500.Load())

	require.Zero(t, errors500.Load(),
		"concurrent team join must produce no 500 errors (got %d)", errors500.Load())

	require.Positive(t, successes.Load(),
		"at least one join must succeed (got 0 - confirm_reset missing or all rejected?)")

	// Membership is stored as team_id on the users table.
	// Verify that all joiners that reported success share the same team_id (no split-brain).
	var distinctTeams int

	joinerIDs := make([]string, len(joiners))
	for i, u := range joiners {
		joinerIDs[i] = u.UserID
	}

	err = testDBPool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT team_id) FROM users
		WHERE id = ANY($1::uuid[]) AND team_id IS NOT NULL
	`, joinerIDs).Scan(&distinctTeams)
	require.NoError(t, err)

	// All successful joiners must be on the same team (≤1 distinct team_id).
	require.LessOrEqual(t, distinctTeams, 1,
		"all joiners must end up on at most 1 team, got %d distinct team_ids", distinctTeams)

	fmt.Printf("[race] PASS: %d concurrent joins -> %d distinct teams (expected ≤1)\n",
		raceConcurrency, distinctTeams)
}

func TestRace_ConcurrentTeamCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent team creation race load test in short mode")
	}

	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)

	ctx := context.Background()
	username := "lt_race_team_user"
	email := "lt_race_team@loadtest.local"
	password := "ValidPass1"

	regBody, err := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	regReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		Fixture.BaseURL+"/api/v1/auth/register",
		strings.NewReader(string(regBody)),
	)
	require.NoError(t, err)
	regReq.Header.Set("Content-Type", "application/json")
	regResp, err := http.DefaultClient.Do(regReq)
	require.NoError(t, err)

	defer regResp.Body.Close()

	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	loginBody, err := json.Marshal(map[string]string{"email": email, "password": password})
	require.NoError(t, err)
	loginReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		Fixture.BaseURL+"/api/v1/auth/login",
		strings.NewReader(string(loginBody)),
	)
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := http.DefaultClient.Do(loginReq)
	require.NoError(t, err)

	defer loginResp.Body.Close()

	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	var loginData struct {
		AccessToken *string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&loginData))
	require.NotNil(t, loginData.AccessToken)
	token := "Bearer " + *loginData.AccessToken

	_, err = testDBPool.Exec(ctx, "UPDATE competition SET mode = 'teams_only', updated_at = NOW() WHERE id = 1")
	require.NoError(t, err)

	defer func() {
		_, _ = testDBPool.Exec(context.Background(), "UPDATE competition SET mode = 'teams_only', updated_at = NOW() WHERE id = 1")
	}()

	concurrency := 20
	statusCounts := make(map[int]int)

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	start := make(chan struct{})

	for range concurrency {
		wg.Go(func() {
			<-start

			body, err := json.Marshal(openapi.CreateSoloTeamRequest{})
			if err != nil {
				mu.Lock()
				statusCounts[0]++
				mu.Unlock()

				return
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodPost,
				Fixture.BaseURL+"/api/v1/teams/solo",
				strings.NewReader(string(body)),
			)
			if err != nil {
				mu.Lock()
				statusCounts[0]++
				mu.Unlock()

				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				mu.Lock()
				statusCounts[0]++
				mu.Unlock()

				return
			}
			defer resp.Body.Close()

			mu.Lock()
			statusCounts[resp.StatusCode]++
			mu.Unlock()
		})
	}

	close(start)
	wg.Wait()

	fmt.Printf("\n[race] Solo team creation results: %v\n", statusCounts)

	require.Equal(t, 1, statusCounts[http.StatusCreated],
		"exactly 1 solo team creation must succeed (got %v)", statusCounts)
}
