package mailer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wahrwelt-kit/go-logkit"
)

// ErrMailerStopped is returned by Send when the AsyncMailer has been stopped.
var ErrMailerStopped = errors.New("mailer stopped")

// AsyncMailer wraps a Mailer and delivers messages asynchronously via a buffered channel and worker pool.
type AsyncMailer struct {
	ctx      context.Context
	delegate Mailer
	msgChan  chan Message
	workers  int
	l        logkit.Logger
	stopped  atomic.Bool
	stopOnce sync.Once
	sendMu   sync.RWMutex
	workerWG sync.WaitGroup
}

// NewAsyncMailer creates an AsyncMailer with the given send-queue buffer size and worker count.
func NewAsyncMailer(
	ctx context.Context,
	delegate Mailer,
	bufferSize int,
	workers int,
	l logkit.Logger,
) *AsyncMailer {
	if ctx == nil {
		panic("mailer.NewAsyncMailer: nil context")
	}

	return &AsyncMailer{
		ctx:      ctx,
		delegate: delegate,
		msgChan:  make(chan Message, bufferSize),
		workers:  workers,
		l:        l,
	}
}

// Start launches the background reader goroutine and worker pool. Must be called before Send.
func (m *AsyncMailer) Start() {
	workerCount := m.workers
	if workerCount <= 0 {
		workerCount = 1
	}

	m.workerWG.Add(workerCount)

	for range workerCount {
		go m.worker()
	}
}

// Stop signals the mailer to shut down, drains any queued messages, and waits for all workers to finish.
func (m *AsyncMailer) Stop() {
	m.stopOnce.Do(func() {
		m.sendMu.Lock()
		m.stopped.Store(true)
		close(m.msgChan)
		m.sendMu.Unlock()

		m.workerWG.Wait()
	})
}

// Send enqueues msg for asynchronous delivery. Returns ErrMailerStopped if Stop has been called,
// or an error if the internal queue is full.
func (m *AsyncMailer) Send(_ context.Context, msg Message) error {
	m.sendMu.RLock()
	defer m.sendMu.RUnlock()

	if m.stopped.Load() {
		return ErrMailerStopped
	}

	select {
	case m.msgChan <- msg:
		return nil
	default:
		return errors.New("mailer.Send: queue is full")
	}
}

func (m *AsyncMailer) worker() {
	defer m.workerWG.Done()

	for msg := range m.msgChan {
		m.send(msg)
	}
}

const sendTimeout = 30 * time.Second

func (m *AsyncMailer) send(msg Message) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(m.ctx), sendTimeout)
	defer cancel()

	err := m.delegate.Send(ctx, msg)
	if err != nil {
		m.l.WithError(err).WithFields(logkit.Fields{"to": msg.To}).Error("AsyncMailer: failed to send email")
	}
}
