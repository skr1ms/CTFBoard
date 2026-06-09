package load_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type AttackResult struct {
	Name    string
	RPS     int
	Metrics *vegeta.Metrics
}

var reportCollector struct {
	mu      sync.Mutex
	results []*AttackResult
}

func RunAttack(attacker *vegeta.Attacker, name string, rps int, duration time.Duration, targeter vegeta.Targeter) *AttackResult {
	rate := vegeta.Rate{Freq: rps, Per: time.Second}
	results := attacker.Attack(targeter, rate, duration, name)

	m := &vegeta.Metrics{}

	for res := range results {
		m.Add(res)
	}

	m.Close()

	r := &AttackResult{Name: name, RPS: rps, Metrics: m}

	if dir := os.Getenv("LOAD_TEST_REPORT_DIR"); dir != "" {
		reportCollector.mu.Lock()
		reportCollector.results = append(reportCollector.results, r)
		reportCollector.mu.Unlock()
	}

	return r
}

func WarmUpAttack(attacker *vegeta.Attacker, rps int, duration time.Duration, targeter vegeta.Targeter) {
	rate := vegeta.Rate{Freq: rps, Per: time.Second}
	results := attacker.Attack(targeter, rate, duration, "warmup")

	for range results {
	}
}

func FlushReports() {
	dir := os.Getenv("LOAD_TEST_REPORT_DIR")
	if dir == "" {
		return
	}

	reportCollector.mu.Lock()
	results := reportCollector.results
	reportCollector.mu.Unlock()

	if len(results) == 0 {
		return
	}

	fname := fmt.Sprintf("report_%s.json", time.Now().UTC().Format("20060102_150405"))

	path := filepath.Join(dir, fname)

	err := WriteJSONReport(path, results)
	if err != nil {
		fmt.Printf("[load-test] warn: write report to %s: %v\n", path, err)

		return
	}

	fmt.Printf("[load-test] report written: %s\n", path)
}

func PrintMetrics(r *AttackResult) {
	m := r.Metrics
	fmt.Printf("\n=== %s @ %d RPS ===\n", r.Name, r.RPS)
	fmt.Printf("  Duration:    %s\n", m.Duration.Round(time.Millisecond))
	fmt.Printf("  Requests:    %d\n", m.Requests)
	fmt.Printf("  Throughput:  %.2f req/s\n", m.Throughput)
	fmt.Printf("  Success:     %.2f%%\n", m.Success*100)
	fmt.Printf("  Latency P50: %s\n", m.Latencies.P50.Round(time.Millisecond))
	fmt.Printf("  Latency P95: %s\n", m.Latencies.P95.Round(time.Millisecond))
	fmt.Printf("  Latency P99: %s\n", m.Latencies.P99.Round(time.Millisecond))
	fmt.Printf("  Latency Max: %s\n", m.Latencies.Max.Round(time.Millisecond))

	if len(m.Errors) > 0 {
		fmt.Printf("  Errors:      %v\n", m.Errors)
	}

	fmt.Printf("  Status codes:\n")

	for code, count := range m.StatusCodes {
		fmt.Printf("    %s: %d\n", code, count)
	}
}

func PrintStepSummary(r *AttackResult) {
	m := r.Metrics
	fmt.Printf("  RPS=%4d  success=%.1f%%  p99=%s  p95=%s\n",
		r.RPS,
		m.Success*100,
		m.Latencies.P99.Round(time.Millisecond),
		m.Latencies.P95.Round(time.Millisecond),
	)
}

func WriteJSONReport(path string, results []*AttackResult) error {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return err
	}

	type jsonEntry struct {
		Name       string        `json:"name"`
		RPS        int           `json:"rps"`
		Requests   uint64        `json:"requests"`
		Success    float64       `json:"success"`
		Throughput float64       `json:"throughput"`
		P50        time.Duration `json:"p50_ms"`
		P95        time.Duration `json:"p95_ms"`
		P99        time.Duration `json:"p99_ms"`
		MaxLatency time.Duration `json:"max_ms"`
	}

	entries := make([]jsonEntry, 0, len(results))
	for _, r := range results {
		m := r.Metrics
		entries = append(entries, jsonEntry{
			Name:       r.Name,
			RPS:        r.RPS,
			Requests:   m.Requests,
			Success:    m.Success,
			Throughput: m.Throughput,
			P50:        m.Latencies.P50 / time.Millisecond,
			P95:        m.Latencies.P95 / time.Millisecond,
			P99:        m.Latencies.P99 / time.Millisecond,
			MaxLatency: m.Latencies.Max / time.Millisecond,
		})
	}

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	err = enc.Encode(entries)
	if err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0o600)
}

func NewAttacker(workers uint64) *vegeta.Attacker {
	return vegeta.NewAttacker(
		vegeta.Workers(workers),
		vegeta.Timeout(30*time.Second),
		vegeta.KeepAlive(true),
		vegeta.MaxConnections(5000),
	)
}
