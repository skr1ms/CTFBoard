package txctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAfterCommitOrNow_WithoutCollectorRunsImmediately(t *testing.T) {
	t.Parallel()

	var called bool

	AfterCommitOrNow(context.Background(), func(context.Context) {
		called = true
	})

	assert.True(t, called)
}

func TestAfterCommitOrNow_WithCollectorDefersUntilRun(t *testing.T) {
	t.Parallel()

	collector := NewCollector()
	ctx := WithCollector(context.Background(), collector)

	var calls []int

	AfterCommitOrNow(ctx, func(context.Context) { calls = append(calls, 1) })
	AfterCommitOrNow(ctx, func(context.Context) { calls = append(calls, 2) })

	assert.Empty(t, calls)

	collector.Run(context.Background())

	assert.Equal(t, []int{1, 2}, calls)
}

func TestCollectorRun_DoesNotRunCallbacksTwice(t *testing.T) {
	t.Parallel()

	collector := NewCollector()
	ctx := WithCollector(context.Background(), collector)

	var calls int

	AfterCommitOrNow(ctx, func(context.Context) { calls++ })

	collector.Run(context.Background())
	collector.Run(context.Background())

	assert.Equal(t, 1, calls)
}

func TestCollectorRun_RecoversCallbackPanicAndContinues(t *testing.T) {
	t.Parallel()

	collector := NewCollector()
	ctx := WithCollector(context.Background(), collector)

	var called bool

	AfterCommitOrNow(ctx, func(context.Context) { panic("boom") })
	AfterCommitOrNow(ctx, func(context.Context) { called = true })

	collector.Run(context.Background())

	assert.True(t, called)
}
