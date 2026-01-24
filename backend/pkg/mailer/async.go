package mailer

import (
	"context"
	"fmt"
	"sync"

	"github.com/skr1ms/CTFBoard/pkg/logger"
)

type AsyncMailer struct {
	delegate Mailer
	msgChan  chan Message
	wg       sync.WaitGroup
	quit     chan struct{}
	workers  int
	l        logger.Logger
}

func NewAsyncMailer(delegate Mailer, bufferSize, workers int, l logger.Logger) *AsyncMailer {
	return &AsyncMailer{
		delegate: delegate,
		msgChan:  make(chan Message, bufferSize),
		quit:     make(chan struct{}),
		workers:  workers,
		l:        l,
	}
}

func (m *AsyncMailer) Start() {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker()
	}
}

func (m *AsyncMailer) Stop() {
	close(m.quit)
	m.wg.Wait()
}

func (m *AsyncMailer) Send(ctx context.Context, msg Message) error {
	select {
	case m.msgChan <- msg:
		return nil
	default:
		return fmt.Errorf("mailer queue is full")
	}
}

func (m *AsyncMailer) worker() {
	defer m.wg.Done()
	for {
		select {
		case msg := <-m.msgChan:
			m.send(msg)
		case <-m.quit:
			// Drain the channel
			for {
				select {
				case msg := <-m.msgChan:
					m.send(msg)
				default:
					return
				}
			}
		}
	}
}

func (m *AsyncMailer) send(msg Message) {
	if err := m.delegate.Send(context.Background(), msg); err != nil {
		m.l.WithError(err).Error(fmt.Sprintf("AsyncMailer: failed to send email to %s", msg.To))
	}
}
