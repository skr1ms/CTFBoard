package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateLinearDynamicScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  int
		min      int
		decay    int
		solves   int
		expected int
	}{
		{"no solves", 500, 100, 20, 0, 500},
		{"1 solve (first blood)", 500, 100, 20, 1, 500},
		{"2 solves", 500, 100, 20, 2, 480},   // 500 - 400/20 * 1 = 500 - 20 = 480
		{"10 solves", 500, 100, 20, 10, 320}, //nolint:gocritic // math annotation, not commented-out code
		{"20 solves", 500, 100, 20, 20, 120}, //nolint:gocritic // math annotation, not commented-out code
		{"21 solves (>= decay)", 500, 100, 20, 21, 100},
		{"100 solves", 500, 100, 20, 100, 100},
		{"decay zero (fallback)", 500, 100, 0, 2, 100},
		{"initial < min", 50, 100, 20, 5, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CalculateLinearDynamicScore(tt.initial, tt.min, tt.decay, tt.solves)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateScore_Dispatch(t *testing.T) {
	t.Parallel()

	logResult := CalculateScore(DecayLogarithmic, 500, 100, 20, 10)
	assert.Equal(t, CalculateDynamicScore(500, 100, 20, 10), logResult)

	linResult := CalculateScore(DecayLinear, 500, 100, 20, 10)
	assert.Equal(t, CalculateLinearDynamicScore(500, 100, 20, 10), linResult)

	defaultResult := CalculateScore("unknown", 500, 100, 20, 10)
	assert.Equal(t, CalculateDynamicScore(500, 100, 20, 10), defaultResult)
}

func TestCalculateDynamicScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  int
		min      int
		decay    int
		solves   int
		expected int
	}{
		{"no solves", 500, 100, 20, 0, 500},
		{"1 solve", 500, 100, 20, 1, 500},                    // First blood (N-1=0) -> Initial
		{"10 solves", 500, 100, 20, 10, 419},                 // 500 - 400/400 * 9^2 = 500 - 81 = 419
		{"20 solves", 500, 100, 20, 20, 139},                 // 500 - 1 * 19^2 = 500 - 361 = 139
		{"100 solves", 500, 100, 20, 100, 100},               // > decay -> Min
		{"decay zero (fallback to 1)", 500, 100, 0, 10, 100}, // > 1 -> Min
		{"high initial", 1000, 100, 50, 50, 136},             // 1000 - 900/2500 * 49^2 = 1000 - 864 = 136
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CalculateDynamicScore(tt.initial, tt.min, tt.decay, tt.solves)
			assert.Equal(t, tt.expected, got)
		})
	}
}
