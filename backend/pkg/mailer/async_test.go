package mailer_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/stretchr/testify/assert"
)

type mockMailer struct {
	mu       sync.Mutex
	messages []mailer.Message
	wg       sync.WaitGroup
}

func (m *mockMailer) Send(ctx context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	m.wg.Done()
	return nil
}

func TestAsyncMailer(t *testing.T) {
	mock := &mockMailer{}
	asyncMailer := mailer.NewAsyncMailer(mock, 10, 1)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	msg := mailer.Message{To: "test@example.com", Subject: "Test", Body: "Body"}

	mock.wg.Add(1)
	err := asyncMailer.Send(context.Background(), msg)
	assert.NoError(t, err)

	// Wait for worker to process
	done := make(chan struct{})
	go func() {
		mock.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for mailer")
	}

	mock.mu.Lock()
	assert.Len(t, mock.messages, 1)
	assert.Equal(t, "test@example.com", mock.messages[0].To)
	mock.mu.Unlock()
}
