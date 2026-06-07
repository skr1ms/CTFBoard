package load_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStress_Registration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registration throughput load test in short mode")
	}

	require.NotNil(t, Fixture)

	targeter := RegisterTargeter(Fixture)

	registrationProfile := []StressStep{
		{RPS: raceScale(10), Duration: 10 * time.Second},
		{RPS: raceScale(30), Duration: 10 * time.Second},
		{RPS: raceScale(50), Duration: 10 * time.Second},
		{RPS: raceScale(100), Duration: 10 * time.Second},
		{RPS: raceScale(200), Duration: 10 * time.Second},
	}

	fmt.Println("\n[stress] Registration - bcrypt + advisory lock throughput:")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "status")

	var results []*AttackResult

	for _, step := range registrationProfile {
		attacker := NewAttacker(200)
		r := RunAttack(attacker, fmt.Sprintf("register@%drps", step.RPS), step.RPS, step.Duration, targeter)
		attacker.Stop()

		results = append(results, r)
		m := r.Metrics

		serverErrors := m.StatusCodes["500"]
		status := "OK"

		if serverErrors > 0 {
			status = fmt.Sprintf("WARN: %d 500s", serverErrors)
		}

		fmt.Printf("  %-6d  %-12.1f  %-10s  %-10s  %s\n",
			step.RPS,
			m.Success*100,
			m.Latencies.P95.Round(time.Millisecond),
			m.Latencies.P99.Round(time.Millisecond),
			status,
		)

		require.Zero(t, serverErrors,
			"registration at %d RPS must not produce 500 errors (got %d)", step.RPS, serverErrors)

		if m.Success < 0.50 {
			fmt.Printf("  [stress] early exit: success %.1f%% below 50%%\n", m.Success*100)

			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.NotEmpty(t, results)
	first := results[0]
	require.GreaterOrEqual(t, first.Metrics.Success, SuccessThreshold,
		"registration at %d RPS must have ≥%.0f%% success (got %.1f%%)",
		registrationProfile[0].RPS, SuccessThreshold*100, first.Metrics.Success*100)
}

func TestStress_RegistrationBurst(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping burst test in -short mode")
	}

	require.NotNil(t, Fixture)

	targeter := RegisterTargeter(Fixture)

	fmt.Println("\n[stress] Registration Burst - thundering-herd spike:")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "500s")

	burstRPS := raceScale(500)

	attacker := NewAttacker(500)
	r := RunAttack(attacker, fmt.Sprintf("register_burst@%drps", burstRPS), burstRPS, 5*time.Second, targeter)
	attacker.Stop()

	m := r.Metrics
	serverErrors := m.StatusCodes["500"]

	fmt.Printf("  %-6d  %-12.1f  %-10s  %-10s  %d\n",
		burstRPS,
		m.Success*100,
		m.Latencies.P95.Round(time.Millisecond),
		m.Latencies.P99.Round(time.Millisecond),
		serverErrors,
	)

	require.Zero(t, serverErrors,
		"registration burst must not produce 500 errors; advisory lock/bcrypt must handle thundering herd")
}
