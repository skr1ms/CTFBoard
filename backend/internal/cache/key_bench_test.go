package cache

import (
	"testing"

	"github.com/google/uuid"
)

func BenchmarkKeyUser(b *testing.B) {
	b.ReportAllocs()

	id := uuid.New().String()

	for b.Loop() {
		_ = KeyUser(id)
	}
}

func BenchmarkKeyTeam(b *testing.B) {
	b.ReportAllocs()

	id := uuid.New().String()

	for b.Loop() {
		_ = KeyTeam(id)
	}
}

func BenchmarkKeyScoreboardBracket(b *testing.B) {
	b.ReportAllocs()

	id := uuid.New().String()

	for b.Loop() {
		_ = KeyScoreboardBracket(id)
	}
}

func BenchmarkKeyScoreboardFrozenAt(b *testing.B) {
	b.ReportAllocs()

	const freezeUnix int64 = 1712000000

	for b.Loop() {
		_ = KeyScoreboardFrozenAt(freezeUnix)
	}
}

func BenchmarkKeyScoreboardBracketFrozenAt(b *testing.B) {
	b.ReportAllocs()

	id := uuid.New().String()

	const freezeUnix int64 = 1712000000

	for b.Loop() {
		_ = KeyScoreboardBracketFrozenAt(id, freezeUnix)
	}
}
