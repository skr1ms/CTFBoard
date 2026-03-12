package competition

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

var (
	BatcherDroppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "submission_batcher_dropped_total",
		Help: "Total number of submissions dropped because the batcher channel was full.",
	})
	BatcherFlushedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "submission_batcher_flushed_total",
		Help: "Total number of submissions successfully flushed to the database.",
	})
	BatcherFlushErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "submission_batcher_flush_errors_total",
		Help: "Total number of individual submission flush failures.",
	})
	batcherMetricsOnce sync.Once
)

func initBatcherMetrics() {
	batcherMetricsOnce.Do(func() {
		prometheus.MustRegister(BatcherDroppedTotal, BatcherFlushedTotal, BatcherFlushErrorsTotal)
	})
}

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
	ch               chan *entity.Submission
	repo             repo.SubmissionRepository
	logger           logger.Logger
	done             chan struct{}
	shutdownFlushCtx chan context.Context
	wg               sync.WaitGroup
	stopped          atomic.Bool
}

var _ usecase.SubmissionBatcher = (*SubmissionBatcher)(nil)

type BatcherOption func(*SubmissionBatcher)

func WithBatcherLogger(l logger.Logger) BatcherOption {
	return func(b *SubmissionBatcher) { b.logger = l }
}

func NewSubmissionBatcher(submissionRepo repo.SubmissionRepository, opts ...BatcherOption) *SubmissionBatcher {
	initBatcherMetrics()
	b := &SubmissionBatcher{
		ch:               make(chan *entity.Submission, defaultChannelBufSize),
		repo:             submissionRepo,
		done:             make(chan struct{}),
		shutdownFlushCtx: make(chan context.Context, 1),
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.logger == nil {
		b.logger = logger.Noop()
	}
	b.wg.Add(1)
	go b.run()
	return b
}

// Enqueue adds a submission to the flush queue. Non-blocking; if the buffer is full
// the submission is written synchronously to avoid data loss.
func (b *SubmissionBatcher) Enqueue(sub *entity.Submission) {
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
	defer b.wg.Done()
	ticker := time.NewTicker(defaultFlushInterval)
	defer ticker.Stop()

	buf := make([]*entity.Submission, 0, defaultBatchSize)
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

func (b *SubmissionBatcher) retryCreate(ctx context.Context, sub *entity.Submission, attempts int) error {
	var lastErr error
	delay := retryCreateBaseDelay
	for i := 0; i < attempts; i++ {
		if err := b.repo.Create(ctx, sub); err != nil {
			lastErr = err
			if i < attempts-1 {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return fmt.Errorf("SubmissionBatcher - retryCreate: %w", ctx.Err())
				case <-timer.C:
					if delay < retryCreateMaxDelay {
						delay *= 2
						if delay > retryCreateMaxDelay {
							delay = retryCreateMaxDelay
						}
					}
				}
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (b *SubmissionBatcher) flush(ctx context.Context, subs []*entity.Submission) {
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
