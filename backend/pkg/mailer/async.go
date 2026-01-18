package mailer

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type AsyncMailer struct {
	delegate Mailer
	msgChan  chan Message
	wg       sync.WaitGroup
	quit     chan struct{}
	workers  int
}

func NewAsyncMailer(delegate Mailer, bufferSize, workers int) *AsyncMailer {
	return &AsyncMailer{
		delegate: delegate,
		msgChan:  make(chan Message, bufferSize),
		quit:     make(chan struct{}),
		workers:  workers,
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
	// Create a background context for sending, as the original request context might be canceled
	if err := m.delegate.Send(context.Background(), msg); err != nil {
		log.Printf("AsyncMailer: failed to send email to %s: %v", msg.To, err)
	}
}
