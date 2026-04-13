package load_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStress_HintUnlockThroughput(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.HintEntries)

	targeter := HintUnlockTargeter(Fixture)

	hintProfile := []StressStep{
		{RPS: raceScale(50), Duration: 10 * time.Second},
		{RPS: raceScale(100), Duration: 10 * time.Second},
		{RPS: raceScale(200), Duration: 10 * time.Second},
		{RPS: raceScale(500), Duration: 10 * time.Second},
	}

	fmt.Println("\n[stress] Hint Unlock - advisory lock + balance deduction under load:")
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "status")

	var results []*AttackResult

	for _, step := range hintProfile {
		attacker := NewAttacker(200)
		r := RunAttack(attacker, fmt.Sprintf("hint_unlock@%drps", step.RPS), step.RPS, step.Duration, targeter)
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
			"hint unlock at %d RPS must not produce 500 errors (advisory lock must prevent races)", step.RPS)

		if m.Success < 0.50 {
			fmt.Printf("  [stress] early exit: success %.1f%% below 50%%\n", m.Success*100)

			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.NotEmpty(t, results)
	first := results[0]
	require.GreaterOrEqual(t, first.Metrics.Success, SuccessThreshold,
		"hint unlock at %d RPS must have ≥%.0f%% success (got %.1f%%)",
		hintProfile[0].RPS, SuccessThreshold*100, first.Metrics.Success*100)
}
