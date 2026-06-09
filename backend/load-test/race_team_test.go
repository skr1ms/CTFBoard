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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestRace_ConcurrentTeamJoin(t *testing.T) {
	require.NotNil(t, Fixture)

	ctx := context.Background()
	client, err := openapi.NewClientWithResponses(Fixture.BaseURL + "/api/v1")
	require.NoError(t, err)

	host := createStandaloneLoadUser(t, ctx, client, "race_join_host")
	hostToken := host.Token
	hostAuth := bearerEditor(hostToken)

	createResp, err := client.PostTeamsWithResponse(ctx, openapi.CreateTeamRequest{
		Name: "rjt_" + uuid.NewString()[:8],
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

	joiners := make([]UserToken, raceConcurrency)
	for i := range raceConcurrency {
		joiners[i] = createStandaloneLoadUser(t, ctx, client, fmt.Sprintf("race_join_%02d", i))
	}

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

func createStandaloneLoadUser(t *testing.T, ctx context.Context, client *openapi.ClientWithResponses, prefix string) UserToken {
	t.Helper()

	suffix := uuid.NewString()
	username := fmt.Sprintf("%s_%s", prefix, suffix[:8])
	email := fmt.Sprintf("%s_%s@loadtest.local", prefix, suffix)
	password := "ValidPass1"

	regResp, err := client.PostAuthRegisterWithResponse(ctx, openapi.PostAuthRegisterJSONRequestBody{
		Username: username,
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, regResp.StatusCode(), "register %s: %s", username, string(regResp.Body))

	loginResp, err := client.PostAuthLoginWithResponse(ctx, openapi.PostAuthLoginJSONRequestBody{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, loginResp.StatusCode(), "login %s: %s", username, string(loginResp.Body))
	require.NotNil(t, loginResp.JSON200)
	require.NotNil(t, loginResp.JSON200.AccessToken)

	token := "Bearer " + *loginResp.JSON200.AccessToken

	meResp, err := client.GetAuthMeWithResponse(ctx, bearerEditor(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, meResp.StatusCode(), "me %s: %s", username, string(meResp.Body))
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)

	return UserToken{UserID: *meResp.JSON200.ID, Token: token}
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

	_, err = testDBPool.Exec(ctx, "UPDATE competition SET mode = 'solo_only', updated_at = NOW() WHERE id = 1")
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
	require.Zero(t, statusCounts[http.StatusInternalServerError],
		"concurrent solo team creation must produce no 500 errors (got %v)", statusCounts)

	for status := range statusCounts {
		switch status {
		case http.StatusCreated, http.StatusBadRequest, http.StatusConflict, http.StatusTooManyRequests:
		default:
			t.Fatalf("unexpected status for concurrent solo team creation: %d (got %v)", status, statusCounts)
		}
	}

	var teamCount int

	err = testDBPool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM teams
		WHERE captain_id = (SELECT id FROM users WHERE email = $1)
		  AND deleted_at IS NULL
	`, email).Scan(&teamCount)
	require.NoError(t, err)
	require.Equal(t, 1, teamCount, "concurrent solo creation must leave exactly one active team")
}
