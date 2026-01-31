package competition

import "testing"

func TestCalculateDynamicScore(t *testing.T) {
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
			got := CalculateDynamicScore(tt.initial, tt.min, tt.decay, tt.solves)
			if got != tt.expected {
				t.Errorf("CalculateDynamicScore(%d, %d, %d, %d) = %d, expected %d",
					tt.initial, tt.min, tt.decay, tt.solves, got, tt.expected)
			}
		})
	}
}
