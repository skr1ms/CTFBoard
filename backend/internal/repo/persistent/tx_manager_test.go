package persistent

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTx struct {
	rollbackErr   error
	rollbackCalls int
}

func (tx *fakeTx) Begin(context.Context) (pgx.Tx, error) {
	panic("unexpected Begin")
}

func (tx *fakeTx) Commit(context.Context) error {
	panic("unexpected Commit")
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
