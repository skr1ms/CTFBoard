package persistent

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
)

type fakeTx struct {
	rollbackErr   error
	rollbackCalls int
	commitErr     error
	commitCalls   int
	commitHook    func()
}

func (tx *fakeTx) Begin(context.Context) (pgx.Tx, error) {
	panic("unexpected Begin")
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.commitCalls++
	if tx.commitHook != nil {
		tx.commitHook()
	}

	return tx.commitErr
}

func (tx *fakeTx) Rollback(context.Context) error {
	tx.rollbackCalls++

	return tx.rollbackErr
}

func (tx *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	panic("unexpected CopyFrom")
}

func (tx *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	panic("unexpected SendBatch")
}

func (tx *fakeTx) LargeObjects() pgx.LargeObjects {
	panic("unexpected LargeObjects")
}

func (tx *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	panic("unexpected Prepare")
}

func (tx *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec")
}

func (tx *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (tx *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

func (tx *fakeTx) Conn() *pgx.Conn {
	panic("unexpected Conn")
}

func TestTransactionManagerFinishTx_RollsBackOnPanic(t *testing.T) {
	t.Parallel()

	tx := &fakeTx{}
	tm := &TransactionManager{}

	err := tm.finishTx(context.Background(), "Run", "boom", false, tx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic: boom")
	assert.Equal(t, 1, tx.rollbackCalls)
}

func TestTransactionManagerFinishTx_RollbackErrorDoesNotMaskBusinessError(t *testing.T) {
	t.Parallel()

	businessErr := errors.New("business failed")
	tx := &fakeTx{rollbackErr: errors.New("rollback failed")}
	tm := &TransactionManager{}

	err := tm.finishTx(context.Background(), "Run", nil, false, tx, businessErr)

	assert.ErrorIs(t, err, businessErr)
	assert.Equal(t, 1, tx.rollbackCalls)
}

func TestTransactionManagerFinishTx_CommittedSkipsRollback(t *testing.T) {
	t.Parallel()

	tx := &fakeTx{}
	tm := &TransactionManager{}

	err := tm.finishTx(context.Background(), "Run", nil, true, tx, nil)

	require.NoError(t, err)
	assert.Zero(t, tx.rollbackCalls)
}

func TestTransactionManagerRun_RunsAfterCommitCallbacksAfterCommit(t *testing.T) {
	t.Parallel()

	var calls int

	tx := &fakeTx{
		commitHook: func() {
			assert.Zero(t, calls)
		},
	}
	tm := &TransactionManager{
		beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return tx, nil
		},
	}

	err := tm.Run(context.Background(), func(ctx context.Context) error {
		txctx.AfterCommitOrNow(ctx, func(context.Context) { calls++ })
		assert.Zero(t, calls)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, tx.commitCalls)
	assert.Zero(t, tx.rollbackCalls)
	assert.Equal(t, 1, calls)
}

func TestTransactionManagerRun_DefersNestedCallbacksUntilOuterCommit(t *testing.T) {
	t.Parallel()

	var (
		calls      []string
		beginCalls int
	)

	tx := &fakeTx{
		commitHook: func() {
			assert.Empty(t, calls)
		},
	}
	tm := &TransactionManager{
		beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			beginCalls++

			return tx, nil
		},
	}

	err := tm.Run(context.Background(), func(ctx context.Context) error {
		txctx.AfterCommitOrNow(ctx, func(context.Context) { calls = append(calls, "outer") })

		err := tm.Run(ctx, func(ctx context.Context) error {
			txctx.AfterCommitOrNow(ctx, func(context.Context) { calls = append(calls, "nested") })

			return nil
		})
		require.NoError(t, err)
		assert.Empty(t, calls)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, beginCalls)
	assert.Equal(t, []string{"outer", "nested"}, calls)
}

func TestTransactionManagerRun_DiscardsAfterCommitCallbacksOnRollback(t *testing.T) {
	t.Parallel()

	businessErr := errors.New("business failed")
	tx := &fakeTx{}
	tm := &TransactionManager{
		beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return tx, nil
		},
	}

	var calls int

	err := tm.Run(context.Background(), func(ctx context.Context) error {
		txctx.AfterCommitOrNow(ctx, func(context.Context) { calls++ })

		return businessErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, businessErr)
	assert.Zero(t, tx.commitCalls)
	assert.Equal(t, 1, tx.rollbackCalls)
	assert.Zero(t, calls)
}

func TestTransactionManagerRunSerializable_DiscardsCallbacksFromFailedRetry(t *testing.T) {
	t.Parallel()

	txs := []*fakeTx{{}, {}}

	var beginCalls int

	tm := &TransactionManager{
		beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			tx := txs[beginCalls]
			beginCalls++

			return tx, nil
		},
	}

	var calls []int

	var attempt int

	err := tm.RunSerializable(context.Background(), func(ctx context.Context) error {
		attempt++
		currentAttempt := attempt

		txctx.AfterCommitOrNow(ctx, func(context.Context) {
			calls = append(calls, currentAttempt)
		})

		if currentAttempt == 1 {
			return &pgconn.PgError{Code: "40001"}
		}

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, beginCalls)
	assert.Equal(t, 1, txs[0].rollbackCalls)
	assert.Zero(t, txs[0].commitCalls)
	assert.Equal(t, 1, txs[1].commitCalls)
	assert.Equal(t, []int{2}, calls)
}
