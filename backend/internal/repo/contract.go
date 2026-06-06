package repo

import (
	"context"

	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
)

// SolveForPointsRecalc is a type alias for scoring.SolveForPointsRecalc kept for
// backwards compatibility with existing repo interface consumers and mocks.
type SolveForPointsRecalc = scoring.SolveForPointsRecalc

// =============================================================================
// Shared
// =============================================================================

type (
	// TransactionManager runs database transactions at various isolation levels (read-write, serializable, read-only).
	TransactionManager interface {
		Run(ctx context.Context, fn func(context.Context) error) error
		RunSerializable(ctx context.Context, fn func(context.Context) error) error
		ReadOnly(ctx context.Context, fn func(context.Context) error) error
	}
)
