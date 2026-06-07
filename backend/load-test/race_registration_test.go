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
)

func TestRace_ConcurrentRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent registration race load test in short mode")
	}

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
