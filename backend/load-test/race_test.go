package load_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const raceConcurrency = 50

func TestRace_FlagSubmission(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.RaceUserToken)
	require.NotEmpty(t, Fixture.RaceChallengeID)

	chalID := Fixture.RaceChallengeID
	correctFlag := Fixture.Flags[len(Fixture.Flags)-1]
	token := Fixture.RaceUserToken

	type result struct {
		status int
		body   string
	}

	results := make([]result, raceConcurrency)

	var wg sync.WaitGroup

	start := make(chan struct{})

	for i := range raceConcurrency {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			<-start

			body, err := json.Marshal(map[string]string{"flag": correctFlag})
			if err != nil {
				results[idx] = result{status: 0, body: err.Error()}

				return
			}

			req, err := http.NewRequestWithContext(context.Background(),
				http.MethodPost,
				Fixture.BaseURL+"/api/v1/challenges/"+chalID+"/submit",
				strings.NewReader(string(body)),
			)
			if err != nil {
				results[idx] = result{status: 0, body: err.Error()}

				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results[idx] = result{status: 0, body: err.Error()}

				return
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				results[idx] = result{status: 0, body: err.Error()}

				return
			}

			results[idx] = result{status: resp.StatusCode, body: string(respBody)}
		}(i)
	}

	close(start)
	wg.Wait()

	statusCounts := make(map[int]int)

	for _, r := range results {
		statusCounts[r.status]++
	}

	fmt.Printf("\n[race] Flag submission results: %v\n", statusCounts)

	// The submit endpoint is idempotent: both the first solve and subsequent
	// correct submissions return 200 {"correct": true}. ErrAlreadySolved is
	// handled internally and never surfaces as 409, so all concurrent submits
	// are expected to receive 200.
	require.Equal(t, raceConcurrency, statusCounts[http.StatusOK],
		"all %d concurrent submits must receive 200 (got %v)", raceConcurrency, statusCounts)

	var solveCount int

	err := testDBPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM solves
		 WHERE challenge_id = (SELECT id FROM challenges WHERE id::text = $1)`,
		chalID,
	).Scan(&solveCount)
	require.NoError(t, err)
	require.Equal(t, 1, solveCount,
		"exactly 1 solve must be recorded for challenge %s despite %d concurrent submits (got %d)",
		chalID, raceConcurrency, solveCount)

	var correctSubmissions int

	err = testDBPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM submissions
		 WHERE challenge_id = (SELECT id FROM challenges WHERE id::text = $1)
		   AND is_correct = true`,
		chalID,
	).Scan(&correctSubmissions)
	require.NoError(t, err)
	require.Equal(t, 1, correctSubmissions,
		"exactly 1 correct submission must be recorded despite %d concurrent submits (got %d)",
		raceConcurrency, correctSubmissions)

	fmt.Printf("[race] PASS: %d concurrent submits -> 1 solve, 1 correct submission in DB\n", raceConcurrency)
}

func TestRace_HintUnlock(t *testing.T) {
	require.NotNil(t, Fixture)

	if Fixture.RaceHintID == "" {
		t.Skip("no hint entries seeded, skipping hint race test")
	}

	token := Fixture.RaceUserToken
	chalID := Fixture.RaceHintChalID
	hintID := Fixture.RaceHintID

	type result struct {
		status int
	}

	results := make([]result, raceConcurrency)

	var wg sync.WaitGroup

	start := make(chan struct{})

	for i := range raceConcurrency {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			<-start

			req, err := http.NewRequestWithContext(context.Background(),
				http.MethodPost,
				Fixture.BaseURL+"/api/v1/challenges/"+chalID+"/hints/"+hintID+"/unlock",
				http.NoBody,
			)
			if err != nil {
				results[idx] = result{status: 0}

				return
			}

			req.Header.Set("Authorization", token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results[idx] = result{status: 0}

				return
			}
			defer resp.Body.Close()

			results[idx] = result{status: resp.StatusCode}
		}(i)
	}

	close(start)
	wg.Wait()

	statusCounts := make(map[int]int)

	for _, r := range results {
		statusCounts[r.status]++
	}

	fmt.Printf("\n[race] Hint unlock results: %v\n", statusCounts)

	require.Positive(t, statusCounts[http.StatusOK]+statusCounts[http.StatusNoContent],
		"at least one hint unlock must succeed")

	var unlockCount int

	err := testDBPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM hint_unlocks
		 WHERE hint_id = (SELECT id FROM hints WHERE id::text = $1)`,
		hintID,
	).Scan(&unlockCount)
	require.NoError(t, err)
	require.Equal(t, 1, unlockCount,
		"exactly 1 hint_unlock row must exist for hint %s despite %d concurrent requests (got %d)",
		hintID, raceConcurrency, unlockCount)

	fmt.Printf("[race] PASS: %d concurrent unlocks -> 1 hint_unlock row in DB\n", raceConcurrency)
}

func TestRace_ConcurrentRegistration(t *testing.T) {
	require.NotNil(t, Fixture)

	ctx := context.Background()
	email := "lt_race_reg_dup@loadtest.local"
	password := "ValidPass1"

	concurrency := raceConcurrency

	var (
		successes atomic.Int32
		conflicts atomic.Int32
		errors500 atomic.Int32
		wg        sync.WaitGroup
	)

	start := make(chan struct{})

	for i := range concurrency {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			<-start

			username := fmt.Sprintf("lt_race_reg_%d", idx)
			body, _ := json.Marshal(map[string]string{
				"username": username,
				"email":    email,
				"password": password,
			})

			req, err := http.NewRequestWithContext(ctx, http.MethodPost,
				Fixture.BaseURL+"/api/v1/auth/register",
				strings.NewReader(string(body)),
			)
			if err != nil {
				return
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}

			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusCreated:
				successes.Add(1)
			case http.StatusConflict:
				conflicts.Add(1)
			case http.StatusInternalServerError:
				errors500.Add(1)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	fmt.Printf("\n[race] Concurrent registration results: 201=%d 409=%d 500=%d\n",
		successes.Load(), conflicts.Load(), errors500.Load())

	require.Zero(t, errors500.Load(),
		"concurrent registration must produce no 500 errors (got %d)", errors500.Load())

	require.Equal(t, int32(1), successes.Load(),
		"exactly 1 registration must succeed with 201 (got %d)", successes.Load())

	var userCount int

	err := testDBPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE email = $1`, email,
	).Scan(&userCount)
	require.NoError(t, err)
	require.Equal(t, 1, userCount,
		"exactly 1 user row must exist for email %s after %d concurrent registrations (got %d)",
		email, concurrency, userCount)

	fmt.Printf("[race] PASS: %d concurrent registrations -> 1 user row in DB\n", concurrency)
}

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
		"at least one join must succeed (got 0 — confirm_reset missing or all rejected?)")

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
