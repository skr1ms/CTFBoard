package mailer_test

import (
	"context"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/mailer/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAsyncMailer(t *testing.T) {
	mockMailer := mocks.NewMockMailer(t)
	l := logger.New(&logger.Options{
		Level:  logger.InfoLevel,
		Output: logger.ConsoleOutput,
	})
	asyncMailer := mailer.NewAsyncMailer(mockMailer, 10, 1, l)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	msg := mailer.Message{To: "test@example.com", Subject: "Test", Body: "Body"}
	done := make(chan struct{})

	mockMailer.On("Send", mock.Anything, mock.MatchedBy(func(m mailer.Message) bool {
		return m.To == "test@example.com" && m.Subject == "Test" && m.Body == "Body"
	})).Return(nil).Once().Run(func(mock.Arguments) { close(done) })

	err := asyncMailer.Send(context.Background(), msg)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for mailer Send")
	}
}
