package persistent

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
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

func (tm *TransactionManager) DB(ctx context.Context) repo.PgxExecer {
	return ExtractDB(ctx, tm.pool)
}

// ExtractDB returns the pgx.Tx embedded in ctx when one is present, otherwise
// returns pool so that it satisfies sqlc.DBTX in both cases.
func ExtractDB(ctx context.Context, pool *pgxpool.Pool) sqlc.DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return pool
}

// Run executes fn inside a ReadCommitted transaction
// If ctx already carries a transaction, fn is called directly (nested call reuses it).
func (tm *TransactionManager) Run(ctx context.Context, fn func(context.Context) error) (retErr error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	return tm.runInNewTx(ctx, "Run", pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite}, fn)
}

// ReadOnly executes fn inside a read-only transaction (no writes)
// If ctx already carries a transaction, fn is called directly
// Use for scoreboard, statistics, and listing operations.
func (tm *TransactionManager) ReadOnly(ctx context.Context, fn func(context.Context) error) (retErr error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	return tm.runInNewTx(ctx, "ReadOnly", pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly}, fn)
}

// runInNewTx opens a new pgx transaction with opts, injects it into a derived
// context (txKey + isoLevelKey), and calls fn. A deferred finishTx handles
// cleanup: it rolls back if fn returned an error or panicked, and does nothing
// if Commit already succeeded (committed flag). The named return retErr allows
// finishTx to inspect and potentially replace the return value (e.g. surfacing
// a Rollback error when fn itself succeeded but commit was never reached).
func (tm *TransactionManager) runInNewTx(ctx context.Context, op string, opts pgx.TxOptions, fn func(context.Context) error) (retErr error) {
	tx, err := tm.pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("TransactionManager - %s - BeginTx: %w", op, err)
	}

	var committed bool

	defer func() {
		p := recover()

		retErr = tm.finishTx(ctx, op, p, committed, tx, retErr)
		if p != nil {
			panic(p)
		}
	}()

	ctxTx := context.WithValue(ctx, txKey{}, tx)

	ctxTx = context.WithValue(ctxTx, isoLevelKey{}, opts.IsoLevel)
	if err := fn(ctxTx); err != nil {
		return fmt.Errorf("TransactionManager - %s - fn: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("TransactionManager - %s - Commit: %w", op, err)
	}

	committed = true

	return nil
}

// finishTx is called from the defer inside runInNewTx. If a panic was recovered
// it wraps the value as an error and returns it (the caller re-panics after).
// Otherwise, if the transaction was not committed, it attempts Rollback using
// context.WithoutCancel so that a cancelled ctx does not suppress the rollback.
// A Rollback error is only returned when retErr is nil, preserving the original
// business error in all other cases. This guarantees that the first error wins
// and cleanup errors do not mask application logic failures.
func (tm *TransactionManager) finishTx(ctx context.Context, op string, p any, committed bool, tx pgx.Tx, retErr error) error {
	if p != nil {
		if pErr, ok := p.(error); ok {
			return fmt.Errorf("TransactionManager - %s - panic: %w", op, pErr)
		}

		return fmt.Errorf("TransactionManager - %s - panic: %v", op, p)
	}

	if !committed {
		err := tx.Rollback(context.WithoutCancel(ctx))
		if err != nil && retErr == nil {
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
		return false, fmt.Errorf("TransactionManager - RunSerializable - nested tx: unknown isolation level")
	}

	if currentIso != pgx.Serializable {
		return false, fmt.Errorf("TransactionManager - RunSerializable - nested tx: existing %s transaction is not serializable", currentIso)
	}

	return true, fn(ctx)
}

const (
	serializableMaxRetries     = 3
	serializableRetryMaxJitter = 10 * time.Millisecond
	serializableRetryBaseDelay = 5 * time.Millisecond
)

// RunSerializable executes fn inside a Serializable transaction
// If ctx already carries a transaction, fn is called only when that transaction is already Serializable
// otherwise an error is returned (isolation level cannot be "upgraded" from ReadCommitted to Serializable)
// New transactions are retried up to serializableMaxRetries times on serialization failure (40001).
func (tm *TransactionManager) RunSerializable(ctx context.Context, fn func(context.Context) error) (retErr error) {
	if ran, err := runSerializableInExistingTx(ctx, fn); ran || err != nil {
		if err != nil {
			return fmt.Errorf("TransactionManager - RunSerializable: %w", err)
		}

		return nil
	}

	opts := pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite}

	for attempt := range serializableMaxRetries {
		err := tm.runInNewTx(ctx, "RunSerializable", opts, fn)
		if err == nil {
			return nil
		}

		if !isSerializationFailure(err) || attempt == serializableMaxRetries-1 {
			return err
		}

		jitter := time.Duration(cryptoRandN(int64(serializableRetryMaxJitter))) + time.Duration(attempt+1)*serializableRetryBaseDelay
		t := time.NewTimer(jitter)

		select {
		case <-ctx.Done():
			t.Stop()

			return ctx.Err()
		case <-t.C:
		}
	}

	return nil
}

// isSerializationFailure reports whether err is a PostgreSQL serialization failure
// (SQLSTATE 40001), which triggers a retry in RunSerializable.
func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "40001" {
		return true
	}

	return false
}

// cryptoRandN returns a cryptographically random int64 in [0, n). Used for
// jitter in the serializable-transaction retry back-off to prevent thundering herd.
func cryptoRandN(n int64) int64 {
	if n <= 0 {
		return 0
	}

	var buf [8]byte

	_, _ = rand.Read(buf[:])

	return int64(binary.LittleEndian.Uint64(buf[:]) % uint64(n))
}
