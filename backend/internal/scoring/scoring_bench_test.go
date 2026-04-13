package scoring_test

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
)

func BenchmarkCalculateDynamicScore_Log(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		scoring.CalculateDynamicScore(500, 100, 10, 50)
	}
}

func BenchmarkCalculateDynamicScore_Linear(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		scoring.CalculateLinearDynamicScore(500, 100, 10, 50)
	}
}

func BenchmarkCalculateScore_Dispatch(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		scoring.CalculateScore(scoring.DecayLogarithmic, 500, 100, 10, 50)
	}
}

func BenchmarkCalculateDynamicScore_HighSolves(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		scoring.CalculateDynamicScore(1000, 50, 3, 500)
	}
}
