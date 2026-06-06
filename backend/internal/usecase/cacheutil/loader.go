package cacheutil

import (
	"context"
	"time"
)

const LoaderTimeout = 10 * time.Second

// LoaderContext preserves request-scoped values but decouples shared cache loaders
// from the first caller's cancellation.
func LoaderContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), LoaderTimeout)
}
