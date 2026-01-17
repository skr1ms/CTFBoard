package usecase

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
		{"1 solve", 500, 100, 20, 1, 486},
		{"10 solves", 500, 100, 20, 10, 382},
		{"20 solves (decay point)", 500, 100, 20, 20, 300},
		{"100 solves", 500, 100, 20, 100, 112},
		{"decay zero (static)", 500, 100, 0, 10, 500},
		{"high initial", 1000, 100, 50, 50, 550},
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
