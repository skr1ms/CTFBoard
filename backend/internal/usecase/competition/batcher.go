package competition

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	batcherDroppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "submission_batcher_dropped_total",
		Help: "Total number of submissions dropped because the batcher channel was full.",
	})
	batcherFlushedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "submission_batcher_flushed_total",
		Help: "Total number of submissions successfully flushed to the database.",
	})
	batcherFlushErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "submission_batcher_flush_errors_total",
		Help: "Total number of individual submission flush failures.",
	})
	batcherMetricsOnce sync.Once
)

func initBatcherMetrics() {
	batcherMetricsOnce.Do(func() {
		prometheus.MustRegister(batcherDroppedTotal, batcherFlushedTotal, batcherFlushErrorsTotal)
	})
}

const (
	defaultBatchSize      = 64
	defaultFlushInterval  = 100 * time.Millisecond
	defaultChannelBufSize = 1024
	flushTimeout          = 10 * time.Second
)

// SubmissionBatcher collects submission log entries into a buffered channel
// and flushes them from a single background goroutine, reducing connection
// pool pressure compared to one goroutine per request.
type SubmissionBatcher struct {
	ch      chan *entity.Submission
	repo    repo.SubmissionRepository
	logger  logger.Logger
	done    chan struct{}
	wg      sync.WaitGroup
	stopped atomic.Bool
}

var _ usecase.SubmissionBatcher = (*SubmissionBatcher)(nil)

type BatcherOption func(*SubmissionBatcher)

func WithBatcherLogger(l logger.Logger) BatcherOption {
	return func(b *SubmissionBatcher) { b.logger = l }
}

func NewSubmissionBatcher(submissionRepo repo.SubmissionRepository, opts ...BatcherOption) *SubmissionBatcher {
	initBatcherMetrics()
	b := &SubmissionBatcher{
		ch:   make(chan *entity.Submission, defaultChannelBufSize),
		repo: submissionRepo,
		done: make(chan struct{}),
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

// Enqueue adds a submission to the flush queue. Non-blocking: drops the
// submission if the buffer is full or if Stop has been called (fire-and-forget semantics).
func (b *SubmissionBatcher) Enqueue(sub *entity.Submission) {
	if b.stopped.Load() {
		batcherDroppedTotal.Inc()
		b.logger.Warn("SubmissionBatcher: batcher stopped, dropping submission")
		return
	}
	select {
	case b.ch <- sub:
	default:
		batcherDroppedTotal.Inc()
		b.logger.Warn("SubmissionBatcher: channel full, dropping submission")
	}
}

// Stop signals the flush goroutine to drain remaining submissions and exit.
// After Stop returns, Enqueue is a no-op (submissions are dropped).
func (b *SubmissionBatcher) Stop() {
	b.stopped.Store(true)
	close(b.done)
	b.wg.Wait()
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
				b.flush(buf)
				buf = buf[:0]
			}
		case <-ticker.C:
			if len(buf) > 0 {
				b.flush(buf)
				buf = buf[:0]
			}
		case <-b.done:
			for {
				select {
				case sub := <-b.ch:
					buf = append(buf, sub)
				default:
					b.flush(buf)
					return
				}
			}
		}
	}
}

func (b *SubmissionBatcher) flush(subs []*entity.Submission) {
	if len(subs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
	defer cancel()
	if err := b.repo.CreateBatch(ctx, subs); err != nil {
		batcherFlushErrorsTotal.Add(float64(len(subs)))
		b.logger.WithError(err).Error("SubmissionBatcher: batch flush failed, falling back to individual inserts")
		for _, sub := range subs {
			if err := b.repo.Create(ctx, sub); err != nil {
				b.logger.WithError(err).Error("SubmissionBatcher: individual flush failed")
			} else {
				batcherFlushedTotal.Inc()
			}
		}
		return
	}
	batcherFlushedTotal.Add(float64(len(subs)))
}
