package ctxutil

import (
	"context"
	"testing"
	"time"
)

const expectedPostCommitTimeout = 30 * time.Second

func TestPostCommitContextSurvivesParentCancelWithDeadline(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()

	ctx, cancel := PostCommitContext(parent)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("expected post-commit context to survive parent cancellation, got %v", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected post-commit context deadline")
	}

	if time.Until(deadline) > expectedPostCommitTimeout {
		t.Fatalf("deadline exceeds post-commit timeout: %s", time.Until(deadline))
	}
}
