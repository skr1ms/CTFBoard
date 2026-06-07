package load_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMixed_RealisticCTFTraffic(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	emails := make([]string, len(Fixture.Users))

	passwords := make([]string, len(Fixture.Users))
	for i := range Fixture.Users {
		emails[i] = fmt.Sprintf("lt_user_%04d@loadtest.local", i)
		passwords[i] = "ValidPass1"
	}

	attacker := NewAttacker(300)
	targeter := MixedCTFTargeter(Fixture, emails, passwords)

	p := DefaultMixedProfile

	if testing.Short() {
		p = MixedProfile{RPS: raceScale(100), Duration: 10 * time.Second}
	}

	fmt.Printf("\n[mixed] Realistic CTF traffic @ %d RPS for %s:\n", p.RPS, p.Duration)

	r := RunAttack(attacker, "mixed_ctf", p.RPS, p.Duration, targeter)
	PrintMetrics(r)

	m := r.Metrics
	require.GreaterOrEqual(t, m.Success, SuccessThreshold,
		"mixed traffic success rate must be ≥ %.0f%% (got %.2f%%)",
		SuccessThreshold*100, m.Success*100)
	require.LessOrEqual(t, m.Latencies.P99, P99Threshold,
		"mixed traffic P99 must be ≤ %s (got %s)",
		P99Threshold, m.Latencies.P99)
}

func TestMixed_PeakHour(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping peak-hour mixed load test in short mode")
	}

	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	emails := make([]string, len(Fixture.Users))

	passwords := make([]string, len(Fixture.Users))
	for i := range Fixture.Users {
		emails[i] = fmt.Sprintf("lt_user_%04d@loadtest.local", i)
		passwords[i] = "ValidPass1"
	}

	attacker := NewAttacker(400)
	targeter := WeightedTargeter([]weightedEntry{
		{40, ChallengeListTargeter(Fixture)},
		{35, SubmitWrongFlagTargeter(Fixture)},
		{15, ScoreboardTargeter(Fixture)},
		{7, LoginTargeter(Fixture, emails, passwords)},
		{3, MeTargeter(Fixture)},
	})

	rps := raceScale(800)
	duration := 30 * time.Second
	fmt.Printf("\n[mixed] Peak hour @ %d RPS for %s:\n", rps, duration)

	r := RunAttack(attacker, "peak_hour", rps, duration, targeter)
	PrintMetrics(r)

	m := r.Metrics
	require.GreaterOrEqual(t, m.Success, SuccessThreshold,
		"peak hour success rate must be ≥ %.0f%% (got %.2f%%)",
		SuccessThreshold*100, m.Success*100)

	peakP99 := effectiveP99ThresholdStrict()
	require.LessOrEqual(t, m.Latencies.P99, peakP99,
		"peak hour P99 must be ≤ %s (got %s)",
		peakP99, m.Latencies.P99)
}
