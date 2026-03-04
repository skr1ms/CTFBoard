package load_test

import "time"

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
