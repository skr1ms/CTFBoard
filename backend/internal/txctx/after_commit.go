package txctx

import (
	"context"
	"sync"
	"time"
)

const postCommitTimeout = 30 * time.Second

type collectorKey struct{}

// AfterCommitFunc is a best-effort side effect that must run only after the
// outermost database transaction commits successfully.
type AfterCommitFunc func(context.Context)

// Collector stores callbacks registered by nested use cases while a transaction
// is open. It is intentionally small and context-scoped.
type Collector struct {
	mu        sync.Mutex
	callbacks []AfterCommitFunc
}

// NewCollector creates an empty after-commit collector.
func NewCollector() *Collector {
	return &Collector{}
}

// WithCollector returns a context carrying c.
func WithCollector(ctx context.Context, c *Collector) context.Context {
	if c == nil {
		return ctx
	}

	return context.WithValue(ctx, collectorKey{}, c)
}

// InTransaction reports whether ctx is inside a transaction managed by this
// package's collector.
func InTransaction(ctx context.Context) bool {
	_, ok := fromContext(ctx)

	return ok
}

// AfterCommitOrNow registers fn for execution after the outermost transaction
// commits. When ctx is not transactional, fn runs immediately with a bounded
// context that survives request cancellation.
func AfterCommitOrNow(ctx context.Context, fn AfterCommitFunc) {
	if fn == nil {
		return
	}

	if c, ok := fromContext(ctx); ok {
		c.add(fn)

		return
	}

	postCtx, cancel := PostCommitContext(ctx)
	defer cancel()

	runSafely(postCtx, fn)
}

// Run executes all currently registered callbacks in FIFO order. The supplied
// ctx must be the outer context, not the transaction context, so callbacks do
// not inherit a finished pgx transaction from context values.
func (c *Collector) Run(ctx context.Context) {
	if c == nil {
		return
	}

	callbacks := c.take()
	if len(callbacks) == 0 {
		return
	}

	postCtx, cancel := PostCommitContext(ctx)
	defer cancel()

	for _, fn := range callbacks {
		runSafely(postCtx, fn)
	}
}

// PostCommitContext preserves request values while bounding work that should
// outlive request cancellation after a successful transaction.
func PostCommitContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), postCommitTimeout)
}

func fromContext(ctx context.Context) (*Collector, bool) {
	c, ok := ctx.Value(collectorKey{}).(*Collector)

	return c, ok && c != nil
}

func (c *Collector) add(fn AfterCommitFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.callbacks = append(c.callbacks, fn)
}

func (c *Collector) take() []AfterCommitFunc {
	c.mu.Lock()
	defer c.mu.Unlock()

	callbacks := c.callbacks
	c.callbacks = nil

	return callbacks
}

func runSafely(ctx context.Context, fn AfterCommitFunc) {
	defer func() {
		_ = recover()
	}()

	fn(ctx)
}
