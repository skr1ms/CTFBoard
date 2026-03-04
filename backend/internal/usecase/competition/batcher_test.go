package competition

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newSub() *entity.Submission {
	return &entity.Submission{ID: uuid.New(), SubmittedFlag: "flag{test}"}
}

func TestSubmissionBatcher_Enqueue_FlushOnTicker(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)

	flushed := make(chan []*entity.Submission, 1)
	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, subs []*entity.Submission) error {
			flushed <- subs
			return nil
		})

	b := NewSubmissionBatcher(repo)
	defer b.Stop()

	b.Enqueue(newSub())

	select {
	case subs := <-flushed:
		assert.Len(t, subs, 1)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("flush did not happen within timeout")
	}
}

func TestSubmissionBatcher_Enqueue_BatchSizeTriggersFlush(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)

	var mu sync.Mutex
	var received []*entity.Submission
	done := make(chan struct{})

	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, subs []*entity.Submission) error {
			mu.Lock()
			received = append(received, subs...)
			if len(received) >= defaultBatchSize {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			mu.Unlock()
			return nil
		}).Maybe()

	b := NewSubmissionBatcher(repo)
	defer b.Stop()

	for range defaultBatchSize {
		b.Enqueue(newSub())
	}

	select {
	case <-done:
		mu.Lock()
		assert.GreaterOrEqual(t, len(received), defaultBatchSize)
		mu.Unlock()
	case <-time.After(500 * time.Millisecond):
		t.Fatal("batch flush did not happen within timeout")
	}
}

func TestSubmissionBatcher_EnqueueAfterStop_Drops(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)
	// expect optional flush of anything queued before Stop
	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(nil).Maybe()

	b := NewSubmissionBatcher(repo)
	b.Stop()

	// After Stop these must not panic and must be silently dropped
	b.Enqueue(newSub())
	b.Enqueue(newSub())
}

func TestSubmissionBatcher_GracefulShutdown_DrainsRemaining(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)

	var mu sync.Mutex
	var totalFlushed int

	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, subs []*entity.Submission) error {
			mu.Lock()
			totalFlushed += len(subs)
			mu.Unlock()
			return nil
		}).Maybe()

	b := NewSubmissionBatcher(repo)

	const n = 5
	for range n {
		b.Enqueue(newSub())
	}

	b.Stop() // must drain before returning

	mu.Lock()
	assert.Equal(t, n, totalFlushed)
	mu.Unlock()
}

func TestSubmissionBatcher_CreateBatch_ErrorFallsBackToIndividual(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)

	batchErr := errors.New("batch insert failed")
	individualDone := make(chan struct{}, 1)

	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(batchErr).Once()
	repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *entity.Submission) error {
			select {
			case individualDone <- struct{}{}:
			default:
			}
			return nil
		}).Once()

	b := NewSubmissionBatcher(repo)

	b.Enqueue(newSub())

	select {
	case <-individualDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("individual fallback did not fire within timeout")
	}

	b.Stop()
}

func TestSubmissionBatcher_ConcurrentEnqueue_NoRace(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)
	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()

	b := NewSubmissionBatcher(repo)

	var wg sync.WaitGroup
	const goroutines = 50
	const perGoroutine = 20

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				b.Enqueue(newSub())
			}
		}()
	}
	wg.Wait()
	b.Stop()
}

func TestSubmissionBatcher_ChannelFull_Drops(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSubmissionRepository(t)

	// Block all flushes so the channel fills up.
	blocked := make(chan struct{})
	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ []*entity.Submission) error {
			<-blocked
			return nil
		}).Maybe()

	b := NewSubmissionBatcher(repo)

	// Overflow the channel (defaultChannelBufSize = 1024)
	for range defaultChannelBufSize + 100 {
		b.Enqueue(newSub())
	}

	close(blocked)
	require.NotPanics(t, func() { b.Stop() })
}
