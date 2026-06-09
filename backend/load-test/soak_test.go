package load_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSoak_MixedTraffic_Extended runs 300 RPS mixed traffic for 10 minutes to detect
// memory leaks, goroutine leaks, connection pool exhaustion, and GC pressure.
// Skipped in -short mode since it takes ~10 minutes.
func TestSoak_MixedTraffic_Extended(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10-minute soak test in -short mode")
	}

	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	emails := make([]string, len(Fixture.Users))
	passwords := make([]string, len(Fixture.Users))

	for i, u := range Fixture.Users {
		emails[i] = fmt.Sprintf("lt_user_%04d@loadtest.local", i)
		passwords[i] = "ValidPass1"
		_ = u
	}

	targeter := MixedCTFTargeter(Fixture, emails, passwords)

	soakRPS := raceScale(300)

	const soakDuration = 10 * time.Minute

	goroutinesBefore := runtime.NumGoroutine()

	var memBefore runtime.MemStats

	runtime.ReadMemStats(&memBefore)

	fmt.Printf("\n[soak] Mixed traffic %d RPS × %s:\n", soakRPS, soakDuration)
	fmt.Printf("  %-6s  %-12s  %-10s  %-10s  %s\n", "RPS", "success%", "p95", "p99", "status")

	attacker := NewAttacker(500)
	r := RunAttack(attacker, fmt.Sprintf("soak_mixed@%drps", soakRPS), soakRPS, soakDuration, targeter)
	attacker.Stop()

	PrintMetrics(r)

	m := r.Metrics
	serverErrors := m.StatusCodes["500"]

	require.Zero(t, serverErrors,
		"soak test must not produce any 500 errors over %s at %d RPS", soakDuration, soakRPS)

	require.GreaterOrEqual(t, m.Success, SuccessThreshold,
		"soak success rate must be ≥ %.0f%% (got %.2f%%)", SuccessThreshold*100, m.Success*100)

	soakP99Threshold := effectiveP99ThresholdStrict()
	require.LessOrEqual(t, m.Latencies.P99, soakP99Threshold,
		"soak P99 must be ≤ %s (got %s)", soakP99Threshold, m.Latencies.P99)

	runtime.GC() //nolint:revive // Leak checks need a stable post-GC runtime snapshot.
	time.Sleep(loadTestServerIdleTimeout + time.Second)
	runtime.GC() //nolint:revive // Let idle HTTP connections drain before measuring heap/goroutines.

	goroutinesAfter := runtime.NumGoroutine()

	var memAfter runtime.MemStats

	runtime.ReadMemStats(&memAfter)

	goroutineDelta := goroutinesAfter - goroutinesBefore

	heapGrowthMB := float64(int64(memAfter.HeapInuse)-int64(memBefore.HeapInuse)) / 1024 / 1024

	fmt.Printf("\n[soak] Goroutine delta: %+d (%d -> %d)\n", goroutineDelta, goroutinesBefore, goroutinesAfter)
	fmt.Printf("[soak] Heap growth: %.1f MB (%d -> %d bytes in-use)\n", heapGrowthMB, memBefore.HeapInuse, memAfter.HeapInuse)

	require.Less(t, goroutineDelta, 100,
		"goroutine leak: grew by %d goroutines over the soak period", goroutineDelta)

	require.Less(t, heapGrowthMB, 50.0,
		"memory leak: heap grew by %.1f MB over the soak period", heapGrowthMB)

	// Check DB connection pool for leaks after the soak.
	ctx := context.Background()

	poolStat := testDBPool.Stat()
	fmt.Printf("[soak] Post-soak pool: acquired=%d idle=%d total=%d\n",
		poolStat.AcquiredConns(), poolStat.IdleConns(), poolStat.TotalConns())

	require.Zero(t, poolStat.AcquiredConns(),
		"post-soak DB pool must have 0 acquired (leaked) connections (got %d)", poolStat.AcquiredConns())

	var activeConns int

	err := testDBPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active'`).Scan(&activeConns)
	if err == nil {
		maxConns := int(testDBPool.Config().MaxConns)
		fmt.Printf("[soak] Post-soak pg_stat_activity active: %d / %d\n", activeConns, maxConns)
		require.LessOrEqual(t, activeConns, maxConns,
			"post-soak active connections must not exceed pool max (%d)", maxConns)
	}
}
