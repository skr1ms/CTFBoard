package persistent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxBase struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewTxBase(pool *pgxpool.Pool) *TxBase {
	return &TxBase{pool: pool, q: sqlc.New(pool)}
}

func (r *TxBase) BeginTx(ctx context.Context) (repo.Transaction, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
}

func (r *TxBase) BeginSerializableTx(ctx context.Context) (repo.Transaction, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
}

func (r *TxBase) RunTransaction(ctx context.Context, fn func(context.Context, repo.Transaction) error) error {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("RunTransaction - BeginTx: %w", err)
	}
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("RunTransaction: invalid transaction type")
	}
	defer func() {
		if p := recover(); p != nil {
			if rbErr := pgxTx.Rollback(ctx); rbErr != nil {
				_ = rbErr
			}
			panic(p)
		}
	}()
	if err := fn(ctx, tx); err != nil {
		if rbErr := pgxTx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("RunTransaction - FnError: %w, RollbackError: %w", err, rbErr)
		}
		return err
	}
	if err := pgxTx.Commit(ctx); err != nil {
		return fmt.Errorf("RunTransaction - Commit: %w", err)
	}
	return nil
}
