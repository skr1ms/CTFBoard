package mailer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"
	"github.com/wahrwelt-kit/go-logkit"
)

var ErrMailerStopped = errors.New("mailer stopped")

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

func (m *AsyncMailer) Start() {
	m.workPool = pool.New().WithMaxGoroutines(m.workers)
	go m.reader()
}

func (m *AsyncMailer) Stop() {
	m.stopOnce.Do(func() {
		m.stopped.Store(true)
		close(m.quit)
		<-m.readerDone
	})
}

func (m *AsyncMailer) Send(_ context.Context, msg Message) error {
	if m.stopped.Load() {
		return ErrMailerStopped
	}
	select {
	case m.msgChan <- msg:
		return nil
	default:
		return fmt.Errorf("mailer queue is full")
	}
}

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

func (m *AsyncMailer) send(msg Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.delegate.Send(ctx, msg); err != nil {
		m.l.WithError(err).WithFields(logkit.Fields{"to": msg.To}).Error("AsyncMailer: failed to send email")
	}
}
