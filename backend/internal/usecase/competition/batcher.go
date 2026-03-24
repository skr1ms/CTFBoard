package competition

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

var (
	BatcherDroppedTotal     = promauto.NewCounter(prometheus.CounterOpts{Name: "submission_batcher_dropped_total", Help: "Total number of submissions dropped because the batcher channel was full."})
	BatcherFlushedTotal     = promauto.NewCounter(prometheus.CounterOpts{Name: "submission_batcher_flushed_total", Help: "Total number of submissions successfully flushed to the database."})
	BatcherFlushErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "submission_batcher_flush_errors_total", Help: "Total number of individual submission flush failures."})
)

const (
	defaultBatchSize      = 64
	defaultFlushInterval  = 100 * time.Millisecond
	defaultChannelBufSize = 1024
	flushTimeout          = 10 * time.Second
	shutdownFlushTimeout  = 30 * time.Second
	enqueueSyncTimeout    = 5 * time.Second
	retryCreateAttempts   = 3
	retryCreateBaseDelay  = 100 * time.Millisecond
	retryCreateMaxDelay   = 2 * time.Second
)

// SubmissionBatcher collects submission log entries into a buffered channel
// and flushes them from a single background goroutine, reducing connection
// pool pressure compared to one goroutine per request.
type SubmissionBatcher struct {
	ch               chan *domain.Submission
	repo             repo.SubmissionRepository
	logger           logkit.Logger
	done             chan struct{}
	shutdownFlushCtx chan context.Context
	wg               sync.WaitGroup
	stopped          atomic.Bool
}

var _ usecase.SubmissionBatcher = (*SubmissionBatcher)(nil)

type BatcherOption func(*SubmissionBatcher)

func WithBatcherLogger(l logkit.Logger) BatcherOption {
	return func(b *SubmissionBatcher) { b.logger = l }
}

func NewSubmissionBatcher(submissionRepo repo.SubmissionRepository, opts ...BatcherOption) *SubmissionBatcher {
	b := &SubmissionBatcher{
		ch:               make(chan *domain.Submission, defaultChannelBufSize),
		repo:             submissionRepo,
		done:             make(chan struct{}),
		shutdownFlushCtx: make(chan context.Context, 1),
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = logkit.Noop()
	}
	b.wg.Go(b.run)
	return b
}

// Enqueue adds a submission to the flush queue. Non-blocking; if the buffer is full
// the submission is written synchronously to avoid data loss.
func (b *SubmissionBatcher) Enqueue(sub *domain.Submission) {
	if b.stopped.Load() {
		BatcherDroppedTotal.Inc()
		b.logger.Warn("SubmissionBatcher: batcher stopped, dropping submission")
		return
	}
	select {
	case b.ch <- sub:
	default:
		ctx, cancel := context.WithTimeout(context.Background(), enqueueSyncTimeout)
		defer cancel()
		if err := b.repo.Create(ctx, sub); err != nil {
			BatcherDroppedTotal.Inc()
			b.logger.WithError(err).Warn("SubmissionBatcher: channel full and sync write failed, dropping submission")
			return
		}
		BatcherFlushedTotal.Inc()
	}
}

// Stop signals the flush goroutine to drain remaining submissions and exit.
// After Stop returns, Enqueue is a no-op (submissions are dropped).
func (b *SubmissionBatcher) Stop() {
	b.stopped.Store(true)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownFlushTimeout)
	b.shutdownFlushCtx <- ctx
	close(b.done)
	b.wg.Wait()
	cancel()
}

func (b *SubmissionBatcher) run() {
	ticker := time.NewTicker(defaultFlushInterval)
	defer ticker.Stop()

	buf := make([]*domain.Submission, 0, defaultBatchSize)
	for {
		select {
		case sub := <-b.ch:
			buf = append(buf, sub)
			if len(buf) >= defaultBatchSize {
				flushCtx, cancel := context.WithTimeout(context.Background(), flushTimeout)
				b.flush(flushCtx, buf)
				cancel()
				buf = buf[:0]
			}
		case <-ticker.C:
			if len(buf) > 0 {
				flushCtx, cancel := context.WithTimeout(context.Background(), flushTimeout)
				b.flush(flushCtx, buf)
				cancel()
				buf = buf[:0]
			}
		case <-b.done:
			for {
				select {
				case sub := <-b.ch:
					buf = append(buf, sub)
				default:
					var flushCtx context.Context
					select {
					case flushCtx = <-b.shutdownFlushCtx:
					default:
						var cancel context.CancelFunc
						flushCtx, cancel = context.WithTimeout(context.Background(), flushTimeout)
						defer cancel()
					}
					b.flush(flushCtx, buf)
					return
				}
			}
		}
	}
}

func (b *SubmissionBatcher) retryCreate(ctx context.Context, sub *domain.Submission, attempts int) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = retryCreateBaseDelay
	bo.MaxInterval = retryCreateMaxDelay
	bo.MaxElapsedTime = 0
	op := func() error {
		if err := ctx.Err(); err != nil {
			return backoff.Permanent(fmt.Errorf("SubmissionBatcher - retryCreate: %w", err))
		}
		if err := b.repo.Create(ctx, sub); err != nil {
			return err
		}
		return nil
	}
	maxRetries := attempts - 1
	if maxRetries < 0 {
		maxRetries = 0
	}
	return backoff.Retry(op, backoff.WithContext(backoff.WithMaxRetries(bo, uint64(maxRetries)), ctx))
}

func (b *SubmissionBatcher) flush(ctx context.Context, subs []*domain.Submission) {
	if len(subs) == 0 {
		return
	}
	if err := b.repo.CreateBatch(ctx, subs); err != nil {
		BatcherFlushErrorsTotal.Add(float64(len(subs)))
		b.logger.WithError(err).Error("SubmissionBatcher: batch flush failed, falling back to individual inserts")
		for _, sub := range subs {
			if err := b.retryCreate(ctx, sub, retryCreateAttempts); err != nil {
				BatcherDroppedTotal.Inc()
				b.logger.WithError(err).Error("SubmissionBatcher: all retries failed, submission lost")
			} else {
				BatcherFlushedTotal.Inc()
			}
		}
		return
	}
	BatcherFlushedTotal.Add(float64(len(subs)))
}
