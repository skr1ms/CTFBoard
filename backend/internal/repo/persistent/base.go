package persistent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type BaseRepo struct {
	pool *pgxpool.Pool
}

func (b *BaseRepo) Q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, b.pool))
}

func (b *BaseRepo) DB(ctx context.Context) sqlc.DBTX {
	return ExtractDB(ctx, b.pool)
}

// AcquireAdvisoryLock acquires a PostgreSQL transaction-scoped advisory lock
// for lockKey (pg_advisory_xact_lock). Must be called inside a transaction.
func (b *BaseRepo) AcquireAdvisoryLock(ctx context.Context, lockKey int64) error {
	if _, err := b.DB(ctx).Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", lockKey); err != nil {
		return fmt.Errorf("BaseRepo - AcquireAdvisoryLock: %w", err)
	}

	return nil
}
