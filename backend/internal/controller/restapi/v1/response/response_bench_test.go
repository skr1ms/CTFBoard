package response

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func BenchmarkTimePtr_NonNil(b *testing.B) {
	b.ReportAllocs()

	t := time.Now()

	for b.Loop() {
		_ = timePtr(&t)
	}
}

func BenchmarkTimePtr_Nil(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = timePtr(nil)
	}
}

func BenchmarkFromChallenge(b *testing.B) {
	b.ReportAllocs()

	c := &domain.Challenge{
		ID:          uuid.New(),
		Title:       "Bench Challenge",
		Description: "A benchmark challenge for measuring response serialisation",
		Category:    "misc",
		Points:      100,
		SolveCount:  42,
		State:       domain.ChallengeStateVisible,
	}

	for b.Loop() {
		_ = FromChallenge(c)
	}
}

func BenchmarkFromChallengeList(b *testing.B) {
	b.ReportAllocs()

	challenges := make([]*domain.Challenge, 50)
	for i := range challenges {
		challenges[i] = &domain.Challenge{
			ID:          uuid.New(),
			Title:       "Challenge",
			Description: "Description",
			Category:    "misc",
			Points:      100,
			SolveCount:  i,
			State:       domain.ChallengeStateVisible,
		}
	}

	for b.Loop() {
		for _, c := range challenges {
			_ = FromChallenge(c)
		}
	}
}
