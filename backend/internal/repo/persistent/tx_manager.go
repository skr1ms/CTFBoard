package persistent

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

type isoLevelKey struct{}

// TransactionManager manages database transactions using context propagation.
type TransactionManager struct {
	pool *pgxpool.Pool
}

var _ repo.TransactionManager = (*TransactionManager)(nil)

// NewTransactionManager creates a TransactionManager backed by the given pool.
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// ExtractDB returns the pgx.Tx embedded in ctx when one is present, otherwise
// returns pool so that it satisfies sqlc.DBTX in both cases.
func ExtractDB(ctx context.Context, pool *pgxpool.Pool) sqlc.DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// Run executes fn inside a ReadCommitted transaction.
// If ctx already carries a transaction, fn is called directly (nested call reuses it).
func (tm *TransactionManager) Run(ctx context.Context, fn func(context.Context) error) (retErr error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	return tm.runInNewTx(ctx, "Run", pgx.ReadCommitted, fn)
}

// runInNewTx starts a new transaction with the given isolation level and runs fn inside it.
func (tm *TransactionManager) runInNewTx(ctx context.Context, op string, isoLevel pgx.TxIsoLevel, fn func(context.Context) error) (retErr error) {
	tx, err := tm.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: isoLevel})
	if err != nil {
		return fmt.Errorf("TransactionManager - %s - BeginTx: %w", op, err)
	}
	var committed bool
	defer func() {
		retErr = tm.finishTx(op, recover(), committed, tx, ctx, retErr)
	}()
	ctxTx := context.WithValue(ctx, txKey{}, tx)
	ctxTx = context.WithValue(ctxTx, isoLevelKey{}, isoLevel)
	if err := fn(ctxTx); err != nil {
		return fmt.Errorf("TransactionManager - %s - fn: %w", op, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("TransactionManager - %s - Commit: %w", op, err)
	}
	committed = true
	return nil
}

// finishTx handles defer: panic wrapping and rollback if not committed.
func (tm *TransactionManager) finishTx(op string, p any, committed bool, tx pgx.Tx, ctx context.Context, retErr error) error {
	if p != nil {
		if pErr, ok := p.(error); ok {
			return fmt.Errorf("TransactionManager - %s - panic: %w", op, pErr)
		}
		return fmt.Errorf("TransactionManager - %s - panic: %v", op, p)
	}
	if !committed {
		if err := tx.Rollback(context.WithoutCancel(ctx)); err != nil && retErr == nil {
			return fmt.Errorf("TransactionManager - %s - Rollback: %w", op, err)
		}
	}
	return retErr
}

// runSerializableInExistingTx returns (true, nil) if fn was run inside an existing Serializable tx,
// (false, nil) if no existing tx, or (false, err) if existing tx has wrong/unknown isolation.
func runSerializableInExistingTx(ctx context.Context, fn func(context.Context) error) (ran bool, err error) {
	_, inTx := ctx.Value(txKey{}).(pgx.Tx)
	if !inTx {
		return false, nil
	}
	currentIso, ok := ctx.Value(isoLevelKey{}).(pgx.TxIsoLevel)
	if !ok {
		return false, fmt.Errorf("TransactionManager: cannot run Serializable inside existing transaction with unknown isolation level")
	}
	if currentIso != pgx.Serializable {
		return false, fmt.Errorf("TransactionManager: cannot run Serializable inside existing %s transaction", currentIso)
	}
	return true, fn(ctx)
}

// RunSerializable executes fn inside a Serializable transaction.
// If ctx already carries a transaction, fn is called only when that transaction is already Serializable;
// otherwise an error is returned (isolation level cannot be "upgraded" from ReadCommitted to Serializable).
func (tm *TransactionManager) RunSerializable(ctx context.Context, fn func(context.Context) error) (retErr error) {
	if ran, err := runSerializableInExistingTx(ctx, fn); ran || err != nil {
		if err != nil {
			return fmt.Errorf("TransactionManager - RunSerializable: %w", err)
		}
		return nil
	}
	return tm.runInNewTx(ctx, "RunSerializable", pgx.Serializable, fn)
}
