package slug_test

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/slug"
)

func BenchmarkMatchPageSlug_Valid(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		slug.MatchPageSlug("my-valid-page-slug")
	}
}

func BenchmarkMatchPageSlug_Invalid(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		slug.MatchPageSlug("Invalid Slug With Spaces!")
	}
}

func BenchmarkMatchPageSlug_Long(b *testing.B) {
	b.ReportAllocs()

	longSlug := "a-very-long-page-slug-that-has-many-segments-and-tests-regex-performance-at-scale"

	for b.Loop() {
		slug.MatchPageSlug(longSlug)
	}
}
