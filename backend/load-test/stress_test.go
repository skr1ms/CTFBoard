package load_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"github.com/wahrwelt-kit/go-logkit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
)

func TestStress_FlagSubmit(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	targeter := SubmitWrongFlagTargeter(Fixture)

	fmt.Println("\n[stress] Flag Submission - ramp to ceiling:")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "status")

	var (
		breakingRPS int
		results     []*AttackResult
	)

	for _, step := range StressProfile {
		attacker := NewAttacker(500)
		r := RunAttack(attacker, fmt.Sprintf("submit_stress@%drps", step.RPS), step.RPS, step.Duration, targeter)
		attacker.Stop()

		results = append(results, r)

		m := r.Metrics
		broken := m.Success < SuccessThreshold || m.Latencies.P99 > P99Threshold
		status := "OK"

		if broken {
			status = "DEGRADED"

			if breakingRPS == 0 {
				breakingRPS = step.RPS
			}
		}

		fmt.Printf("  %-6d  %-12.1f  %-10s  %-10s  %s\n",
			step.RPS,
			m.Success*100,
			m.Latencies.P95.Round(time.Millisecond),
			m.Latencies.P99.Round(time.Millisecond),
			status,
		)

		if m.Success < 0.50 {
			fmt.Printf("  [stress] early exit: success rate %.1f%% below 50%% at %d RPS\n", m.Success*100, step.RPS)

			break
		}

		time.Sleep(1 * time.Second)
	}

	if breakingRPS > 0 {
		fmt.Printf("\n[stress] Breaking point identified at %d RPS\n", breakingRPS)
	} else {
		fmt.Printf("\n[stress] System sustained all steps without degradation\n")
	}

	require.NotEmpty(t, results)
	first := results[0]
	require.GreaterOrEqual(t, first.Metrics.Success, SuccessThreshold,
		"submit must sustain %d RPS with ≥ %.0f%% success (got %.2f%%)",
		StressProfile[0].RPS, SuccessThreshold*100, first.Metrics.Success*100)
}

func TestStress_BruteForceThroughput(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	targeter := BruteForceTargeter(Fixture, len(Fixture.Users)-1, 0)

	bruteProfile := []StressStep{
		{RPS: 100, Duration: 10 * time.Second},
		{RPS: 500, Duration: 10 * time.Second},
		{RPS: 1000, Duration: 10 * time.Second},
		{RPS: 2000, Duration: 10 * time.Second},
	}

	fmt.Println("\n[stress] BruteForce single-challenge throughput (rate limiter disabled):")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "status")

	for _, step := range bruteProfile {
		attacker := NewAttacker(500)
		r := RunAttack(attacker, fmt.Sprintf("brute_throughput@%drps", step.RPS), step.RPS, step.Duration, targeter)
		attacker.Stop()

		m := r.Metrics
		serverErr := m.StatusCodes["500"]
		status := "OK"

		if serverErr > 0 {
			status = fmt.Sprintf("WARN: %d 500s", serverErr)
		}

		fmt.Printf("  %-6d  %-12.1f  %-10s  %-10s  %s\n",
			step.RPS,
			m.Success*100,
			m.Latencies.P95.Round(time.Millisecond),
			m.Latencies.P99.Round(time.Millisecond),
			status,
		)

		require.Zero(t, serverErr,
			"brute-force at %d RPS must not produce 500 errors", step.RPS)

		if step.RPS <= 1500 {
			require.LessOrEqual(t, m.Latencies.P99, P99Threshold,
				"brute-force P99 must be ≤ %s at %d RPS (got %s)",
				P99Threshold, step.RPS, m.Latencies.P99)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func TestStress_BruteForceRateLimited(t *testing.T) {
	require.NotNil(t, testRedisClient)

	const (
		submitLimit = 10
		window      = time.Minute
		testRPS     = 100
		testDur     = 5 * time.Second
	)

	ctx := context.Background()

	keys, _ := testRedisClient.Keys(ctx, "limiter:brute_rl_test:*").Result() //nolint:errcheck // test cleanup: best-effort
	if len(keys) > 0 {
		testRedisClient.Del(ctx, keys...)
	}

	log, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)

	limited := restapimiddleware.CombinedRateLimit(testRedisClient, []restapimiddleware.RateLimitSpec{
		{
			KeyPrefix: "brute_rl_test:user",
			Limit:     submitLimit,
			Window:    window,
			KeyFunc: func(_ *http.Request) (string, error) {
				return "brute_single_user", nil
			},
		},
	}, log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":0")
	require.NoError(t, err)

	srv := &http.Server{Handler: limited, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}

	go srv.Serve(listener)  //nolint:errcheck // test server
	defer srv.Shutdown(ctx) //nolint:errcheck // best-effort cleanup

	port := listener.Addr().(*net.TCPAddr).Port //nolint:errcheck
	url := fmt.Sprintf("http://localhost:%d/submit", port)

	targeter := func(tgt *vegeta.Target) error {
		tgt.Method = http.MethodPost
		tgt.URL = url
		tgt.Header = http.Header{"Content-Type": {"application/json"}}
		tgt.Body = []byte(`{"flag":"BRUTE{test}"}`)

		return nil
	}

	attacker := NewAttacker(50)
	r := RunAttack(attacker, "brute_force_rate_limited", testRPS, testDur, targeter)
	attacker.Stop()

	m := r.Metrics
	count429 := m.StatusCodes["429"]
	count200 := m.StatusCodes["200"]
	serverErr := m.StatusCodes["500"]

	fmt.Printf("\n[stress] BruteForce rate-limited (limit=%d/min, %d RPS for %s):\n", submitLimit, testRPS, testDur)
	fmt.Printf("  Requests: %d  200s: %d  429s: %d  500s: %d  p95: %s  p99: %s\n",
		m.Requests, count200, count429, serverErr,
		m.Latencies.P95.Round(time.Millisecond),
		m.Latencies.P99.Round(time.Millisecond))

	require.Zero(t, serverErr, "rate-limited brute-force must not produce 500 errors")
	require.LessOrEqual(t, count200, submitLimit+2,
		"expected at most %d successful requests (limit=%d), got %d", submitLimit+2, submitLimit, count200)
	require.Positive(t, count429,
		"expected 429 responses from rate limiter, got none")
	require.InDelta(t, float64(m.Requests), float64(count200+count429), 5,
		"all responses should be either 200 or 429")
}

func TestStress_ChallengeList(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)

	targeter := ChallengeListTargeter(Fixture)

	readProfile := []StressStep{
		{RPS: 100, Duration: 10 * time.Second},
		{RPS: 500, Duration: 10 * time.Second},
		{RPS: 1000, Duration: 10 * time.Second},
		{RPS: 2000, Duration: 10 * time.Second},
		{RPS: 3000, Duration: 10 * time.Second},
	}

	fmt.Println("\n[stress] Challenge List - read scalability:")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s\n", "RPS", "success%", "p95", "p99")

	var results []*AttackResult

	for _, step := range readProfile {
		attacker := NewAttacker(500)
		r := RunAttack(attacker, fmt.Sprintf("challenges_stress@%drps", step.RPS), step.RPS, step.Duration, targeter)
		attacker.Stop()

		results = append(results, r)
		m := r.Metrics
		fmt.Printf("  %-6d  %-12.1f  %-10s  %-10s\n",
			step.RPS,
			m.Success*100,
			m.Latencies.P95.Round(time.Millisecond),
			m.Latencies.P99.Round(time.Millisecond),
		)

		if m.Success < 0.50 {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.NotEmpty(t, results)
	first := results[0]
	require.GreaterOrEqual(t, first.Metrics.Success, 0.99,
		"challenge list at %d RPS must have ≥ 99%% success", readProfile[0].RPS)
}
