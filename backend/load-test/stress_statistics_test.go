package load_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStress_StatisticsEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping statistics throughput load test in short mode")
	}

	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)

	targeter := StatisticsTargeter(Fixture)

	statsProfile := []StressStep{
		{RPS: raceScale(50), Duration: 10 * time.Second},
		{RPS: raceScale(200), Duration: 10 * time.Second},
		{RPS: raceScale(500), Duration: 10 * time.Second},
		{RPS: raceScale(1000), Duration: 10 * time.Second},
	}

	fmt.Println("\n[stress] Statistics endpoints - aggregate query scalability:")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "status")

	var results []*AttackResult

	for _, step := range statsProfile {
		attacker := NewAttacker(500)
		r := RunAttack(attacker, fmt.Sprintf("statistics@%drps", step.RPS), step.RPS, step.Duration, targeter)
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
			"statistics at %d RPS must not produce 500 errors", step.RPS)

		if m.Success < 0.50 {
			fmt.Printf("  [stress] early exit: success %.1f%% below 50%%\n", m.Success*100)

			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.NotEmpty(t, results)
	first := results[0]
	require.GreaterOrEqual(t, first.Metrics.Success, SuccessThreshold,
		"statistics at %d RPS must have ≥%.0f%% success (got %.1f%%)",
		statsProfile[0].RPS, SuccessThreshold*100, first.Metrics.Success*100)
}
