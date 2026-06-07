package load_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

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
