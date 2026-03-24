package persistent

import (
	"context"

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
