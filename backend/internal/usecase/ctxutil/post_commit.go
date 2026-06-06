package ctxutil

import (
	"context"
	"time"
)

const postCommitTimeout = 30 * time.Second

// PostCommitContext preserves request values while bounding work that should
// outlive the request cancellation after a successful transaction.
func PostCommitContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), postCommitTimeout)
}
