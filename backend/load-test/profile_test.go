package load_test

import (
	"time"
)

// raceScale reduces RPS by 10x when the race detector is enabled. The race
// detector slows bcrypt by ~400x and testcontainers Postgres by ~5x, making
// high-RPS thresholds unreachable. Under race we still validate correctness
// (zero 500s, ≥99% success) at lower throughput.
func raceScale(rps int) int {
	if raceEnabled {
		if rps/10 < 1 {
			return 1
		}

		return rps / 10
	}

	return rps
}

// effectiveP99Threshold returns a relaxed P99 threshold when the race detector
// is active (testcontainers + instrumentation adds significant latency).
func effectiveP99Threshold() time.Duration {
	if raceEnabled {
		return 2 * time.Second
	}

	return P99Threshold
}

// effectiveP99ThresholdStrict is effectiveP99Threshold for spike/peak assertions.
func effectiveP99ThresholdStrict() time.Duration {
	if raceEnabled {
		return 5 * time.Second
	}

	return P99ThresholdStrict
}

// effectiveSpikeProfile returns SpikeProfile with RPS scaled for race mode.
func effectiveSpikeProfile() []SpikeStep {
	if !raceEnabled {
		return SpikeProfile
	}

	scaled := make([]SpikeStep, len(SpikeProfile))
	for i, s := range SpikeProfile {
		scaled[i] = SpikeStep{RPS: raceScale(s.RPS), Duration: s.Duration}
	}

	return scaled
}

// effectiveStressProfile returns StressProfile with RPS scaled for race mode.
func effectiveStressProfile() []StressStep {
	if !raceEnabled {
		return StressProfile
	}

	scaled := make([]StressStep, len(StressProfile))
	for i, s := range StressProfile {
		scaled[i] = StressStep{RPS: raceScale(s.RPS), Duration: s.Duration}
	}

	return scaled
}

type SpikeStep struct {
	RPS      int
	Duration time.Duration
}

var SpikeProfile = []SpikeStep{
	{RPS: 50, Duration: 10 * time.Second},
	{RPS: 150, Duration: 10 * time.Second},
	{RPS: 300, Duration: 10 * time.Second},
	{RPS: 500, Duration: 10 * time.Second},
	{RPS: 800, Duration: 10 * time.Second},
	{RPS: 1200, Duration: 10 * time.Second},
	{RPS: 1500, Duration: 10 * time.Second},
}

type StressStep struct {
	RPS      int
	Duration time.Duration
}

var StressProfile = []StressStep{
	{RPS: 50, Duration: 15 * time.Second},
	{RPS: 100, Duration: 15 * time.Second},
	{RPS: 200, Duration: 15 * time.Second},
	{RPS: 500, Duration: 15 * time.Second},
	{RPS: 1000, Duration: 15 * time.Second},
	{RPS: 1500, Duration: 15 * time.Second},
	{RPS: 2000, Duration: 15 * time.Second},
}

type MixedProfile struct {
	RPS      int
	Duration time.Duration
}

var DefaultMixedProfile = MixedProfile{
	RPS:      500,
	Duration: 60 * time.Second,
}

const (
	SuccessThreshold   = 0.99
	P99Threshold       = 200 * time.Millisecond
	P99ThresholdStrict = 500 * time.Millisecond
)
