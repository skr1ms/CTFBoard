package mailer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"
	"github.com/wahrwelt-kit/go-logkit"
)

// ErrMailerStopped is returned by Send when the AsyncMailer has been stopped.
var ErrMailerStopped = errors.New("mailer stopped")

// AsyncMailer wraps a Mailer and delivers messages asynchronously via a buffered channel and worker pool.
type AsyncMailer struct {
	delegate   Mailer
	msgChan    chan Message
	quit       chan struct{}
	readerDone chan struct{}
	workers    int
	l          logkit.Logger
	stopped    atomic.Bool
	stopOnce   sync.Once
	workPool   *pool.Pool
}

// NewAsyncMailer creates an AsyncMailer with the given send-queue buffer size and worker count.
func NewAsyncMailer(
	delegate Mailer,
	bufferSize int,
	workers int,
	l logkit.Logger,
) *AsyncMailer {
	return &AsyncMailer{
		delegate:   delegate,
		msgChan:    make(chan Message, bufferSize),
		quit:       make(chan struct{}),
		readerDone: make(chan struct{}),
		workers:    workers,
		l:          l,
	}
}

// Start launches the background reader goroutine and worker pool. Must be called before Send.
func (m *AsyncMailer) Start() {
	m.workPool = pool.New().WithMaxGoroutines(m.workers)
	go m.reader()
}

// Stop signals the mailer to shut down, drains any queued messages, and waits for all workers to finish.
func (m *AsyncMailer) Stop() {
	m.stopOnce.Do(func() {
		m.stopped.Store(true)
		close(m.quit)
		<-m.readerDone
	})
}

// Send enqueues msg for asynchronous delivery. Returns ErrMailerStopped if Stop has been called,
// or an error if the internal queue is full.
func (m *AsyncMailer) Send(_ context.Context, msg Message) error {
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

// reader is the core dispatch goroutine. It forwards messages from msgChan to the worker pool.
// On receiving a quit signal, it drains the remaining buffered messages before waiting for all
// workers to complete and closing readerDone to unblock Stop.
func (m *AsyncMailer) reader() {
	defer close(m.readerDone)

	for {
		select {
		case msg := <-m.msgChan:
			mmsg := msg

			m.workPool.Go(func() { m.send(mmsg) })
		case <-m.quit:
			for {
				select {
				case msg := <-m.msgChan:
					mmsg := msg

					m.workPool.Go(func() { m.send(mmsg) })
				default:
					m.workPool.Wait()

					return
				}
			}
		}
	}
}

const sendTimeout = 30 * time.Second

func (m *AsyncMailer) send(msg Message) {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	err := m.delegate.Send(ctx, msg)
	if err != nil {
		m.l.WithError(err).WithFields(logkit.Fields{"to": msg.To}).Error("AsyncMailer: failed to send email")
	}
}
