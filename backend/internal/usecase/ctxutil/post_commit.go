package ctxutil

import (
	"context"

	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
)

// PostCommitContext preserves request values while bounding work that should
// outlive the request cancellation after a successful transaction.
func PostCommitContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return txctx.PostCommitContext(ctx)
}
