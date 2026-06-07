package load_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSpike_CompetitionStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping competition-start spike load test in short mode")
	}

	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)

	chalTargeter := ChallengeListTargeter(Fixture)
	scoreTargeter := ScoreboardTargeter(Fixture)

	type stepResult struct {
		step       SpikeStep
		chalResult *AttackResult
		sbResult   *AttackResult
	}

	var stepResults []stepResult

	fmt.Println("\n[spike] Competition Start - challenge list + scoreboard ramp:")

	for _, step := range effectiveSpikeProfile() {
		chalAttacker := NewAttacker(200)
		chalR := RunAttack(chalAttacker, fmt.Sprintf("challenges@%drps", step.RPS), step.RPS/2, step.Duration, chalTargeter)
		chalAttacker.Stop()

		sbAttacker := NewAttacker(200)
		sbR := RunAttack(sbAttacker, fmt.Sprintf("scoreboard@%drps", step.RPS), step.RPS/2, step.Duration, scoreTargeter)
		sbAttacker.Stop()

		PrintStepSummary(chalR)
		PrintStepSummary(sbR)
		stepResults = append(stepResults, stepResult{step, chalR, sbR})

		time.Sleep(500 * time.Millisecond)
	}

	var (
		totalRequests  uint64
		totalSuccesses float64
	)

	for _, sr := range stepResults {
		for _, r := range []*AttackResult{sr.chalResult, sr.sbResult} {
			totalRequests += r.Metrics.Requests
			totalSuccesses += float64(r.Metrics.Requests) * r.Metrics.Success
		}
	}

	overallSuccess := totalSuccesses / float64(totalRequests)
	require.GreaterOrEqual(t, overallSuccess, 0.99,
		"overall success rate across spike steps must be ≥ 99%% (got %.2f%%)", overallSuccess*100)

	peak := stepResults[len(stepResults)-1]
	for _, r := range []*AttackResult{peak.chalResult, peak.sbResult} {
		PrintMetrics(r)

		threshold := effectiveP99ThresholdStrict()
		require.LessOrEqual(t, r.Metrics.Latencies.P99, threshold,
			"%s P99 latency exceeded %s at peak RPS (got %s)",
			r.Name, threshold, r.Metrics.Latencies.P99)
	}
}

func TestSpike_EndOfContest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-of-contest spike load test in short mode")
	}

	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	targeter := SubmitWrongFlagTargeter(Fixture)

	fmt.Println("\n[spike] End-of-Contest - flag submission burst:")

	var all []*AttackResult

	for _, step := range effectiveSpikeProfile() {
		attacker := NewAttacker(300)
		r := RunAttack(attacker, fmt.Sprintf("submit@%drps", step.RPS), step.RPS, step.Duration, targeter)
		attacker.Stop()
		PrintStepSummary(r)
		all = append(all, r)

		time.Sleep(500 * time.Millisecond)
	}

	var (
		reqs uint64
		succ float64
	)

	for _, r := range all {
		reqs += r.Metrics.Requests
		succ += float64(r.Metrics.Requests) * r.Metrics.Success
	}

	overall := succ / float64(reqs)
	require.GreaterOrEqual(t, overall, SuccessThreshold,
		"submit spike overall success must be ≥ %.0f%% (got %.2f%%)", SuccessThreshold*100, overall*100)

	PrintMetrics(all[len(all)-1])
}
