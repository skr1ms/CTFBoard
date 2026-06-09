package repo

import "context"

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
