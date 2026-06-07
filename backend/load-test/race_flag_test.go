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
)

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
