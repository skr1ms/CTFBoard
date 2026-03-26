package load_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
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

	validResponses := statusCounts[http.StatusOK] + statusCounts[http.StatusConflict]
	require.Equal(t, raceConcurrency, validResponses,
		"all %d concurrent submits must receive 200 or 409 (got %v)", raceConcurrency, statusCounts)

	require.Equal(t, 1, statusCounts[http.StatusOK],
		"exactly 1 submit should succeed with 200 (got %d)", statusCounts[http.StatusOK])

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

	fmt.Printf("[race] PASS: %d concurrent submits -> 1 solve in DB\n", raceConcurrency)
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
